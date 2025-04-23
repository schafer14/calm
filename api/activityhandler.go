package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"
	"timeapi/data"

	"timeapi/authz"

	casbin "github.com/casbin/casbin/v3"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/segmentio/ksuid"
	"go.etcd.io/bbolt"
)

type userAuthCtx int

var (
	activityBucketName        = []byte("activities")
	currentActivityBucketName = []byte("current_activity")
	currentActivityKey        = "/current"

	userAuthCtxKey userAuthCtx = 1
)

type Activity struct {
	ID                        string   `json:"id"`
	Username                  string   `json:"username"`
	Name                      string   `json:"name"`
	MindPlatterCategory       []string `json:"mind_platter_categories"`
	Categories                []string `json:"categories"`
	ExpectedDurationInMinutes int      `json:"expected_duration_in_minutes"`
	StartTime                 int      `json:"posix_start_time"`
	EndTime                   int      `json:"posix_end_time"`
	Duration                  int      `json:"duration_in_minutes"`
	DoNotDisturb              bool     `json:"do_not_disturb"`
}

func NewActivityHandler(db *bbolt.DB, webAuthn *webauthn.WebAuthn, logger *slog.Logger) http.Handler {

	err := db.Update(func(tx *bbolt.Tx) error {
		tx.CreateBucketIfNotExists(activityBucketName)
		tx.CreateBucketIfNotExists(currentActivityBucketName)
		return nil
	})
	if err != nil {
		logger.Error("creating database", "err", err)
		panic(err)
	}

	stores := data.NewStores(db)

	e := &enforcer{
		enforcer: authz.NewBbolt(db),
		stores:   stores,
		logger:   logger,
	}

	mux := http.NewServeMux()

	handle := mux.Handle
	handleAuth := func(s string, h http.Handler) {
		mux.Handle(s, e.Authed(h))
	}
	handleAuthz := func(s string, h http.Handler) {
		mux.Handle(s, e.Authz(h))
	}

	handle("/", handleIndex())
	handle("/robots.txt", handleRobots())
	handle("/manifest.json", handleManifest())
	// addStatic will use the file system in dev and a prebuilt
	// binary in dev.
	addStatic(mux)

	handleAuth("GET /activities/current", currentActivity(db, logger))
	handleAuth("GET /activities", listActivities(db, logger))
	handleAuth("POST /activities", createActivity(db, logger))
	handleAuth("POST /activities/end", endActivity(db, logger))

	handleAuth("POST /registration/token", newToken(webAuthn, stores, logger))
	handle("POST /registration/end", endRegistration(webAuthn, stores, logger))
	handle("POST /registration/begin/{registrationTokenID}", beginTokenRegistration(webAuthn, stores, logger))

	handle("POST /login/begin", beginLogin(webAuthn, stores, logger))
	handle("POST /login/end", endLogin(webAuthn, stores, logger))

	handleAuth("GET /credentials", listCredentials())
	handleAuth("DELETE /credentials", deleteCredential(stores, logger))

	handleAuthz("POST /users", createUser(stores, logger))
	handleAuthz("GET /users", listUsers(stores, logger))
	handleAuthz("POST /users/{username}/new-device-token", createRegistrationToken(stores, logger))

	return mux

}

func (e *enforcer) Authed(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("Authorization")
		if len(tokenHeader) < 7 || !strings.EqualFold(tokenHeader[:7], "bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error": "invalid bearer token"}`)
			return
		}
		bearer := tokenHeader[7:]

		user, err := e.stores.FindToken(bearer)
		if err != nil {
			e.logger.WarnContext(r.Context(), "unauthorised", "err", err)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error": "invalid bearer token"}`)
			return
		}

		ctx := context.WithValue(r.Context(), userAuthCtxKey, user)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthedUser(r *http.Request) data.Agent {
	return r.Context().Value(userAuthCtxKey).(data.Agent)
}

type enforcer struct {
	enforcer *casbin.Enforcer
	stores   *data.Stores
	logger   *slog.Logger
}

func (e *enforcer) Authz(h http.Handler) http.Handler {
	authz := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := AuthedUser(r)
		resource := r.URL.Path
		action := r.Method

		ok, err := e.enforcer.Enforce(user.Username, resource, action)
		if err != nil {
			e.logger.WarnContext(r.Context(), "enforcing authorisation", "err", err)
			http.Error(w, "enforcing authorisation", http.StatusInternalServerError)
			return
		}

		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})

	return e.Authed(authz)
}

// staticResource is for permissions that do not have a specific entity associated with them.
// For example create a user has no user, where as add role to user 1 does have an entity.
func staticResource(resource string) func(r *http.Request) string {
	return func(r *http.Request) string {
		return resource
	}
}

func handleIndex() http.HandlerFunc {
	f, err := static.ReadFile("static/index.html")
	if err != nil {
		panic(fmt.Errorf("Reading index.html : %w", err))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(f)
	}
}

func handleRobots() http.HandlerFunc {
	f, err := static.ReadFile("static/robots.txt")
	if err != nil {
		panic(fmt.Errorf("Reading index.html : %w", err))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write(f)
	}
}

func handleManifest() http.HandlerFunc {
	f, err := static.ReadFile("static/manifest.json")
	if err != nil {
		panic(fmt.Errorf("Reading index.html : %w", err))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write(f)
	}
}

func createActivity(db *bbolt.DB, logger *slog.Logger) http.HandlerFunc {

	type NewActivity struct {
		Name                      string   `json:"name"`
		MindPlatterCategory       []string `json:"mind_platter_category"`
		Categories                []string `json:"categories"`
		ExpectedDurationInMinutes int      `json:"expected_duration_in_minutes"`
		StartTime                 int      `json:"posix_start_time"`
		DoNotDisturb              bool     `json:"do_not_disturb"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		user := AuthedUser(r)

		var newActivity NewActivity
		newActivity.Categories = []string{}
		newActivity.MindPlatterCategory = []string{}
		newActivity.StartTime = int(time.Now().Unix())

		err := json.NewDecoder(r.Body).Decode(&newActivity)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid json request body", "err", err)
			http.Error(w, "decoding request body", http.StatusBadRequest)
			return
		}

		id := ksuid.New().String()
		activity := Activity{
			ID:                        id,
			Name:                      newActivity.Name,
			MindPlatterCategory:       newActivity.MindPlatterCategory,
			Categories:                newActivity.Categories,
			ExpectedDurationInMinutes: newActivity.ExpectedDurationInMinutes,
			StartTime:                 newActivity.StartTime,
			DoNotDisturb:              newActivity.DoNotDisturb,
			Username:                  user.Username,
		}

		data, err := json.Marshal(activity)
		if err != nil {
			logger.InfoContext(r.Context(), "marshalling data", "err", err)
			http.Error(w, "internal server errror", http.StatusInternalServerError)
			return
		}

		err = db.Update(func(tx *bbolt.Tx) error {
			currentActivityBucket := tx.Bucket(currentActivityBucketName)

			if len(currentActivityBucket.Get([]byte(user.Username+currentActivityKey))) != 0 {
				return fmt.Errorf("cannot create an activity while another activity is unfinished")
			}

			err := currentActivityBucket.Put([]byte(user.Username+currentActivityKey), data)
			if err != nil {
				return fmt.Errorf("saving current activity : %w", err)
			}
			return nil
		})
		if err != nil {
			logger.InfoContext(r.Context(), "accessing data base", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(id)

	}

}

func listActivities(db *bbolt.DB, logger *slog.Logger) http.HandlerFunc {

	type Body struct {
		IntervalStartTime time.Time `json:"interval_start_time"`
		IntervalEndTime   time.Time `json:"interval_end_time"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		var payload Body
		payload.IntervalStartTime = time.Time{}
		payload.IntervalEndTime = time.Now()

		user := AuthedUser(r)

		var activities []Activity

		err := db.View(func(tx *bbolt.Tx) error {
			activityBucket := tx.Bucket(activityBucketName)
			return activityBucket.ForEach(func(k, v []byte) error {

				var activity Activity
				err := json.Unmarshal(v, &activity)
				if err != nil {
					return fmt.Errorf("unmashalling activity : %w", err)
				}

				if int(payload.IntervalStartTime.Unix()) <= activity.StartTime && activity.EndTime <= int(payload.IntervalEndTime.Unix()) {
					if activity.Username == user.Username {
						activities = append(activities, activity)
					}
				}

				return nil
			})
		})
		if err != nil {
			logger.InfoContext(r.Context(), "accessing data base", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(activities)

	}

}

func currentActivity(db *bbolt.DB, logger *slog.Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user := AuthedUser(r)
		var activity Activity

		err := db.View(func(tx *bbolt.Tx) error {
			currentActivityBucket := tx.Bucket(currentActivityBucketName)

			data := currentActivityBucket.Get([]byte(user.Username + currentActivityKey))
			if len(data) > 0 {
				err := json.Unmarshal(data, &activity)
				return err
			}

			return nil
		})
		if err != nil {
			logger.InfoContext(r.Context(), "accessing data base", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if reflect.ValueOf(activity).IsZero() {
			fmt.Fprintf(w, `{}`)
			return
		}

		json.NewEncoder(w).Encode(activity)

	}

}

func endActivity(db *bbolt.DB, logger *slog.Logger) http.HandlerFunc {

	type Body struct {
		EndTime int `json:"posix_end_time"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		user := AuthedUser(r)
		var payload Body
		payload.EndTime = int(time.Now().Unix())

		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			logger.InfoContext(r.Context(), "decoding request body", "err", err)
			http.Error(w, "parsing request body", http.StatusBadRequest)
			return
		}

		err = db.Update(func(tx *bbolt.Tx) error {
			currentActivityBucket := tx.Bucket(currentActivityBucketName)
			activityBucket := tx.Bucket(activityBucketName)

			current := currentActivityBucket.Get([]byte(user.Username + currentActivityKey))
			if current == nil {
				return fmt.Errorf("no current activity found")
			}
			err := currentActivityBucket.Put([]byte(user.Username+currentActivityKey), nil)
			if err != nil {
				return fmt.Errorf("updating current key : %w", err)
			}

			var activity Activity
			err = json.Unmarshal(current, &activity)
			if err != nil {
				return fmt.Errorf("unmarshalling stored activity : %w", err)
			}

			activity.EndTime = payload.EndTime
			activity.Duration = (payload.EndTime - activity.StartTime) / 60

			bytes, err := json.Marshal(activity)
			if err != nil {
				return fmt.Errorf("saving new activity : %w", err)
			}

			return activityBucket.Put([]byte(activity.ID), bytes)
		})
		if err != nil {
			logger.InfoContext(r.Context(), "accessing data base", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

	}

}

func beginTokenRegistration(webAuthn *webauthn.WebAuthn, stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	type Body struct {
		CredentialName string `json:"credential_name"`
	}
	return func(w http.ResponseWriter, r *http.Request) {

		var payload Body
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			logger.ErrorContext(r.Context(), "decoding json body", "err", err)
			http.Error(w, "decoding json body", 400)
			return
		}

		sessionID := rand.Text()
		tokenID := r.PathValue("registrationTokenID")
		tokenInfo, err := stores.ConsumeRegistrationToken(tokenID)
		if err != nil {
			logger.ErrorContext(r.Context(), "getting token info", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}
		user, err := stores.FindUser(tokenInfo)
		if err != nil {
			logger.ErrorContext(r.Context(), "getting user", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}
		cookie := &http.Cookie{
			Name:  "reg-session",
			Value: sessionID,
			Path:  "/",
		}
		http.SetCookie(w, cookie)

		authSelect := protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationPreferred,
		}

		exclusions := []protocol.CredentialDescriptor{}
		for _, cred := range user.Credentials {
			exclusions = append(exclusions, cred.Credential.Descriptor())
		}

		options, session, err := webAuthn.BeginRegistration(user,
			webauthn.WithAuthenticatorSelection(authSelect),
			webauthn.WithExclusions(exclusions),
		)
		if err != nil {
			logger.ErrorContext(r.Context(), "beginning registration", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}

		stores.SaveRegistrationSession(sessionID, data.RegistrationSession{
			Username:    user.Username,
			DeviceName:  payload.CredentialName,
			SessionData: session,
		})

		json.NewEncoder(w).Encode(options)
	}
}

func newToken(webAuthn *webauthn.WebAuthn, stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := AuthedUser(r)
		token := rand.Text()
		err := stores.CreateRegistrationToken(token, user.Username)
		if err != nil {
			logger.WarnContext(r.Context(), "creating registration token", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}

		fmt.Fprintf(w, `{"registration_token":%q}`, token)

	}
}

func createRegistrationToken(stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		username := r.PathValue("username")
		token := rand.Text()
		err := stores.CreateRegistrationToken(token, username)
		if err != nil {
			logger.WarnContext(r.Context(), "creating registration token", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}

		fmt.Fprintf(w, `{"registration_token":%q}`, token)

	}
}

func endRegistration(webAuthn *webauthn.WebAuthn, stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("reg-session")
		if err != nil {
			http.Error(w, "session not found", 404)
			return
		}
		sessionID := cookie.Value

		session, err := stores.FindRegistrationSession(sessionID)
		if err != nil {
			logger.WarnContext(r.Context(), "no registration session found", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		user, err := stores.FindUser(session.Username)
		if err != nil {
			logger.WarnContext(r.Context(), "no registration user found", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		credential, err := webAuthn.FinishRegistration(user, *session.SessionData, r)
		if err != nil {
			logger.WarnContext(r.Context(), "finishing registration", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		user.Credentials = append(user.Credentials, data.Credential{
			Label:      session.DeviceName,
			Credential: *credential,
			LastAuthed: time.Now(),
		})
		err = stores.SaveCredential(credential, user.Username)
		if err != nil {
			logger.WarnContext(r.Context(), "saving credential", "err", err)
			http.Error(w, "unable to register", 500)
			return
		}

		err = stores.UpdateUser(user.Username, user)
		if err != nil {
			logger.WarnContext(r.Context(), "saving user", "err", err)
			http.Error(w, "unable to register", 500)
			return
		}

		json.NewEncoder(w).Encode("success")
	}
}

func beginLogin(webAuthn *webauthn.WebAuthn, stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		sessionID := rand.Text()
		cookie := &http.Cookie{
			Name:  "login-session",
			Value: sessionID,
			Path:  "/",
		}
		http.SetCookie(w, cookie)

		options, session, err := webAuthn.BeginDiscoverableLogin()
		if err != nil {
			logger.WarnContext(r.Context(), "beginning login", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		err = stores.SaveSession(sessionID, session)
		if err != nil {
			logger.WarnContext(r.Context(), "saving session", "err", err)
			http.Error(w, "unable to register", 500)
			return
		}

		json.NewEncoder(w).Encode(options)
	}
}

func endLogin(webAuthn *webauthn.WebAuthn, stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("login-session")
		if err != nil {
			http.Error(w, "session not found", 404)
			return
		}
		sessionID := cookie.Value

		session, err := stores.FindSession(sessionID)
		if err != nil {
			logger.WarnContext(r.Context(), "no registration session found", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		var user data.Agent
		credential, err := webAuthn.FinishDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
			var err error
			user, err = stores.FindUserByCredentialID(rawID)
			return user, err
		}, *session, r)
		if err != nil {
			logger.WarnContext(r.Context(), "finishing login", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		tokenID := rand.Text()
		expires := time.Now().Add(24 * time.Hour)
		token := data.Token{
			UserID:    string(user.Username),
			ExpiresAt: expires,
		}

		// TODO: tokenID could be hashed
		err = stores.SaveToken(tokenID, token)
		if err != nil {
			logger.WarnContext(r.Context(), "creating token", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		for i, cred := range user.Credentials {
			if bytes.Equal(cred.Credential.ID, credential.ID) {
				user.Credentials[i].Credential = *credential
				user.Credentials[i].LastAuthed = time.Now()
			}
		}
		err = stores.UpdateUser(user.Username, user)
		if err != nil {
			logger.WarnContext(r.Context(), "creating token", "err", err)
			http.Error(w, "unable to register", 400)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"access_token": tokenID,
			"expires_at":   expires.Unix(),
			"name":         user.Name,
			"username":     user.Username,
		})
	}
}

func listUsers(stores *data.Stores, logger *slog.Logger) http.HandlerFunc {

	type Body struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		raw, err := stores.ListUsers()
		if err != nil {
			logger.InfoContext(r.Context(), "listing users", "err", err)
			http.Error(w, "listing users", 500)
			return
		}

		users := []Body{}
		for _, user := range raw {
			users = append(users, Body{
				Username: user.Username,
				Name:     user.Name,
			})
		}

		json.NewEncoder(w).Encode(users)
	}
}

func createUser(stores *data.Stores, logger *slog.Logger) http.HandlerFunc {

	type Body struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {

		var payload Body
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid json request body", "err", err)
			http.Error(w, "decoding request body", http.StatusBadRequest)
			return
		}

		if payload.Name == "" {
			payload.Name = payload.Username
		}

		user := data.Agent{
			ID:          data.GenId(),
			Username:    payload.Username,
			Name:        payload.Name,
			Credentials: []data.Credential{},
		}

		username, err := stores.SaveUser(payload.Username, user)
		if err != nil {
			logger.WarnContext(r.Context(), "saving user", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}

		token := rand.Text()
		err = stores.CreateRegistrationToken(token, username)
		if err != nil {
			logger.WarnContext(r.Context(), "creating registration token", "err", err)
			http.Error(w, "internal server error", 500)
			return
		}

		fmt.Fprintf(w, `{"registration_token":%q}`, token)
	}

}

func listCredentials() http.HandlerFunc {
	type res struct {
		ID         string                            `json:"id"`
		LastAuthed int                               `json:"last_authed"`
		Label      string                            `json:"label"`
		Transport  []protocol.AuthenticatorTransport `json:"transport"`
		Flags      webauthn.CredentialFlags          `json:"flags"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		user := AuthedUser(r)

		creds := []res{}
		for _, cred := range user.Credentials {
			creds = append(creds, res{
				ID:         fmt.Sprintf("%x", cred.Credential.ID),
				LastAuthed: int(cred.LastAuthed.Unix()),
				Label:      cred.Label,
				Transport:  cred.Credential.Transport,
				Flags:      cred.Credential.Flags,
			})

		}

		json.NewEncoder(w).Encode(creds)
	}
}

func deleteCredential(stores *data.Stores, logger *slog.Logger) http.HandlerFunc {
	type Body struct {
		ID string `json:"credential_id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := AuthedUser(r)

		var payload Body
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			logger.InfoContext(r.Context(), "invalid json request body", "err", err)
			http.Error(w, "decoding request body", http.StatusBadRequest)
			return
		}

		id, err := hex.DecodeString(payload.ID)
		if err != nil {
			logger.InfoContext(r.Context(), "decoding credential id", "err", err)
			http.Error(w, "decoding credential id", http.StatusBadRequest)
			return
		}

		var thisCred data.Credential
		newCreds := []data.Credential{}
		for _, cred := range user.Credentials {
			if bytes.Equal(cred.Credential.ID, id) {
				thisCred = cred
			} else {
				newCreds = append(newCreds, cred)
			}
		}

		if len(thisCred.Credential.ID) == 0 {
			http.Error(w, "credential not found", 400)
			return
		}

		user.Credentials = newCreds
		err = stores.UpdateUser(user.Username, user)
		if err != nil {
			logger.WarnContext(r.Context(), "deleting user credentail", "err", err)
			http.Error(w, "could not delete credential", 500)
			return
		}

		err = stores.DeleteCredential(id)
		if err != nil {
			logger.WarnContext(r.Context(), "deleting credentail", "err", err)
			http.Error(w, "could not delete credential", 500)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
