package data

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"go.etcd.io/bbolt"
)

var (
	userBucketName                = []byte("users")
	loginSessionsBucketName       = []byte("login_session")
	registrationSessionBucketName = []byte("registration_session")
	tokensBucketName              = []byte("access_tokens")
	credentialsBucketName         = []byte("credentials")
	registrationTokensBucketName  = []byte("registration_tokens")
)

type Agent struct {
	ID          []byte
	Username    string
	Name        string
	Credentials []Credential
}

type Token struct {
	UserID    string
	ExpiresAt time.Time
}

func (u Agent) WebAuthnID() []byte          { return u.ID }
func (u Agent) WebAuthnName() string        { return u.Name }
func (u Agent) WebAuthnDisplayName() string { return u.Username }
func (u Agent) WebAuthnCredentials() []webauthn.Credential {
	creds := []webauthn.Credential{}
	for _, cred := range u.Credentials {
		creds = append(creds, cred.Credential)
	}

	return creds
}

type Credential struct {
	Label      string
	Credential webauthn.Credential
	LastAuthed time.Time
}

type UserByCredentialID struct {
	ID         []byte
	Credential *webauthn.Credential
	Username   string
}

func GenId() []byte {
	token := make([]byte, 64)
	rand.Read(token)
	return token
}

type RegistrationSession struct {
	Username    string
	DeviceName  string
	SessionData *webauthn.SessionData
}

func NewStores(db *bbolt.DB) *Stores {

	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(userBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(tokensBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(loginSessionsBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(credentialsBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(registrationTokensBucketName)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(registrationSessionBucketName)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return &Stores{db}
}

type Stores struct {
	db *bbolt.DB
}

func (s *Stores) SaveUser(username string, u Agent) (string, error) {

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(userBucketName)

		existing := bucket.Get([]byte(username))
		if len(existing) == 0 {
			bytes, err := json.Marshal(u)
			if err != nil {
				panic(err)
			}
			return bucket.Put([]byte(username), bytes)
		}

		for i := range 100 {
			newusername := fmt.Sprintf("%s_%d", username, i)

			existing = bucket.Get([]byte(newusername))
			if len(existing) == 0 {
				u.Username = newusername
				if u.Name == username {
					u.Name = newusername
				}
				bytes, err := json.Marshal(u)
				if err != nil {
					panic(err)
				}
				username = newusername
				return bucket.Put([]byte(newusername), bytes)
			}
		}

		return fmt.Errorf("no available usernames found for : %s", username)

	})
	if err != nil {
		return "", err
	}

	return username, nil
}

func (s *Stores) UpdateUser(username string, u Agent) error {
	bytes, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(userBucketName)

		return bucket.Put([]byte(username), bytes)
	})
}

func (s *Stores) DeleteUser(username string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(userBucketName)

		return bucket.Delete([]byte(username))
	})
}

func (s *Stores) ListUsers() ([]Agent, error) {
	var users []Agent
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(userBucketName)

		bucket.ForEach(func(k, v []byte) error {
			var user Agent
			err := json.Unmarshal(v, &user)
			if err != nil {
				return err
			}
			users = append(users, user)

			return nil
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Stores) FindUser(username string) (Agent, error) {
	var user Agent

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(userBucketName)

		raw := bucket.Get([]byte(username))
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}

		return json.Unmarshal(raw, &user)
	})
	if err != nil {
		return Agent{}, err
	}

	return user, nil
}

func (s *Stores) SaveCredential(cred *webauthn.Credential, userID string) error {
	bytes, err := json.Marshal(UserByCredentialID{
		ID:         cred.ID,
		Credential: cred,
		Username:   userID,
	})
	if err != nil {
		panic(err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(credentialsBucketName)

		return bucket.Put(cred.ID, bytes)
	})
}

func (s *Stores) FindUserByCredentialID(credentialID []byte) (Agent, error) {
	var user Agent

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(credentialsBucketName)
		userBucket := tx.Bucket(userBucketName)

		raw := bucket.Get(credentialID)
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}

		var cred UserByCredentialID
		err := json.Unmarshal(raw, &cred)
		if err != nil {
			return err
		}

		raw = userBucket.Get([]byte(cred.Username))
		if len(raw) == 0 {
			return fmt.Errorf("user not found")
		}

		return json.Unmarshal(raw, &user)
	})
	if err != nil {
		return Agent{}, err
	}

	return user, nil
}

func (s *Stores) DeleteCredential(credentialID []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(credentialsBucketName)

		ex := bucket.Get(credentialID)
		if len(ex) == 0 {
			return fmt.Errorf("credential does not exist")
		}

		return bucket.Delete(credentialID)
	})

}

func (s *Stores) SaveSession(sessionID string, session *webauthn.SessionData) error {
	bytes, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(loginSessionsBucketName)

		return bucket.Put([]byte(sessionID), bytes)
	})
}

func (s *Stores) FindSession(sessionID string) (*webauthn.SessionData, error) {
	var sess *webauthn.SessionData

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(loginSessionsBucketName)

		raw := bucket.Get([]byte(sessionID))
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}

		bucket.Delete([]byte(sessionID))
		return json.Unmarshal(raw, &sess)
	})
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (s *Stores) SaveRegistrationSession(sessionID string, session RegistrationSession) error {
	bytes, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(registrationSessionBucketName)

		return bucket.Put([]byte(sessionID), bytes)
	})
}

func (s *Stores) FindRegistrationSession(sessionID string) (RegistrationSession, error) {
	var sess RegistrationSession

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(registrationSessionBucketName)

		raw := bucket.Get([]byte(sessionID))
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}

		bucket.Delete([]byte(sessionID))
		return json.Unmarshal(raw, &sess)
	})
	if err != nil {
		return RegistrationSession{}, err
	}

	return sess, nil
}

func (s *Stores) SaveToken(tokenID string, token Token) error {
	bytes, err := json.Marshal(token)
	if err != nil {
		panic(err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(tokensBucketName)

		return bucket.Put([]byte(tokenID), bytes)
	})
}

func (s *Stores) FindToken(tokenID string) (Agent, error) {
	var user Agent

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(tokensBucketName)
		userBucket := tx.Bucket(userBucketName)

		raw := bucket.Get([]byte(tokenID))
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}

		var token Token
		err := json.Unmarshal(raw, &token)
		if err != nil {
			return err
		}

		raw = userBucket.Get([]byte(token.UserID))
		if len(raw) == 0 {
			return fmt.Errorf("user not found")
		}

		return json.Unmarshal(raw, &user)
	})
	if err != nil {
		return Agent{}, err
	}

	return user, nil
}

func (s *Stores) CreateRegistrationToken(token string, username string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(registrationTokensBucketName)

		return bucket.Put([]byte(token), []byte(username))
	})
}

func (s *Stores) ConsumeRegistrationToken(token string) (string, error) {
	var userID string
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(registrationTokensBucketName)

		raw := bucket.Get([]byte(token))
		if len(raw) == 0 {
			return fmt.Errorf("not found")
		}
		userID = string(raw)

		return bucket.Delete([]byte(token))
	})
	if err != nil {
		return "", err
	}

	return userID, nil
}
