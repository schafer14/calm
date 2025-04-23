port module Activities exposing (main)

import Ant.Icon as Icon
import Ant.Icons as Icons
import Browser
import Browser.Dom as Dom
import Browser.Navigation as Navigation
import Element as E
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import EverySet
import Http
import Json.Decode as Decode
import Json.Encode as Encode
import Theme.Colors as Colors
import Time
import Url
import Url.Parser as Parser exposing ((</>))


version : String
version =
    "1.0-alpha"



-- MAIN


main : Program InitOpt Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        , onUrlRequest = LinkClicked
        , onUrlChange = UrlChanged
        }



-- MODEL


type ModelState
    = Activity (Loading ActivityModel)
    | Register RegistrationPage
    | Login LoginPage
    | Past (Loading ())
    | Future (Loading ())
    | Settings
    | Version
    | NotFound


type alias Window =
    { width : Int
    , height : Int
    , device : E.Device
    }


type alias InitOpt =
    { width : Int
    , height : Int
    , user : Maybe User
    }


type alias Model =
    { state : ModelState
    , window : Window
    , palette : Colors.Palette
    , user : Maybe User
    , key : Navigation.Key
    }


init : InitOpt -> Url.Url -> Navigation.Key -> ( Model, Cmd Msg )
init startOpt url key =
    let
        ( state, cmds ) =
            gotoRoute startOpt.user <| Url.toString url
    in
    ( { state = state
      , window = { width = startOpt.width, height = startOpt.height, device = E.classifyDevice startOpt }
      , palette = Colors.frappe
      , user = startOpt.user
      , key = key
      }
    , cmds
    )


routeParser : Maybe User -> Parser.Parser (ModelState -> a) a
routeParser user =
    case user of
        Nothing ->
            Parser.oneOf
                [ Parser.map (Login ShowLoginButton) Parser.top
                , Parser.map (Register << RequestingCredentialName "") (Parser.s "register" </> Parser.string)
                , Parser.map Version (Parser.s "version")
                ]

        Just _ ->
            Parser.oneOf
                [ Parser.map (Activity Loading) Parser.top
                , Parser.map Version (Parser.s "version")
                , Parser.map (Activity Loading) (Parser.s "present")
                , Parser.map Settings (Parser.s "settings")
                , Parser.map (Past Loading) (Parser.s "past")
                , Parser.map (Future Loading) (Parser.s "future")
                ]


toRoute : Maybe User -> String -> ModelState
toRoute user string =
    case Url.fromString string of
        Nothing ->
            NotFound

        Just url ->
            Maybe.withDefault NotFound (Parser.parse (routeParser user) url)


gotoRoute : Maybe User -> String -> ( ModelState, Cmd Msg )
gotoRoute maybeUser url =
    let
        nextState =
            toRoute maybeUser url
    in
    case nextState of
        Activity _ ->
            case maybeUser of
                Just user ->
                    ( nextState, fetchCurrentActivity user )

                Nothing ->
                    ( nextState, Cmd.none )

        _ ->
            ( nextState, Cmd.none )



-- UPDATE


type Msg
    = UpdateStartActivityForm StartActivityMsg
    | SubmitStartActivityForm StartActivityModel
    | StartActivity StartActivityModel
    | GotStartActivityResponse (Result Http.Error String)
    | NewViewport Dom.Viewport
    | GotCurrentActivity (Result Http.Error (Maybe CurrentActivity))
    | EndActivity
    | GotEndActivityResponse (Result Http.Error ())
    | RegistrationEvent RegistrationMsg
    | RegistrationComplete (Result Http.Error ())
    | LoginEvent LoginMsg
    | LoginComplete (Result Http.Error User)
    | ClearCache
    | CheckToken Time.Posix
    | LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | Logout


port clearCache : () -> Cmd msg


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ClearCache ->
            ( model, clearCache () )

        CheckToken now ->
            case model.user of
                Just user ->
                    if Time.posixToMillis now > (user.expires_at * 1000 - 300000) then
                        ( { model | user = Nothing }, Cmd.none )

                    else
                        ( model, Cmd.none )

                Nothing ->
                    ( model, clearCache () )

        StartActivity form ->
            ( setState model <| Activity <| Loaded <| NewActivityForm <| form, Cmd.none )

        UpdateStartActivityForm e ->
            case model.state of
                Activity (Loaded (NewActivityForm form)) ->
                    ( setState model <| Activity <| Loaded <| NewActivityForm <| startActivityUpdate form e, Cmd.none )

                _ ->
                    ( model, Cmd.none )

        SubmitStartActivityForm form ->
            case model.user of
                Just user ->
                    ( model, submitStartActivityForm form user )

                Nothing ->
                    ( setState model <| Login ShowLoginButton, Cmd.none )

        NewViewport viewport ->
            let
                w =
                    floor viewport.viewport.width

                h =
                    floor viewport.viewport.height

                device =
                    E.classifyDevice { width = w, height = h }
            in
            ( { model | window = { height = h, width = w, device = device } }, Cmd.none )

        GotStartActivityResponse (Ok _) ->
            case model.user of
                Just user ->
                    ( setState model <| Activity Loading, fetchCurrentActivity user )

                Nothing ->
                    ( setState model <| Login ShowLoginButton, Cmd.none )

        GotStartActivityResponse (Err _) ->
            ( model, Cmd.none )

        GotCurrentActivity (Ok (Just currentActivity)) ->
            ( setState model <| Activity <| Loaded <| ViewCurrentActivity currentActivity, Cmd.none )

        GotCurrentActivity (Ok Maybe.Nothing) ->
            ( setState model <| Activity <| Loaded ActivityHomePage, Cmd.none )

        GotCurrentActivity (Err _) ->
            ( model, Cmd.none )

        EndActivity ->
            case model.user of
                Just user ->
                    ( model, endActivity user )

                Nothing ->
                    ( setState model <| Login ShowLoginButton, Cmd.none )

        GotEndActivityResponse (Ok _) ->
            ( setState model <| Activity <| Loaded ActivityHomePage, Cmd.none )

        GotEndActivityResponse (Err _) ->
            ( model, Cmd.none )

        RegistrationEvent event ->
            case model.state of
                Register page ->
                    let
                        ( newPage, cmd ) =
                            updateRegistrationPage page event
                    in
                    ( setState model <| Register newPage, cmd )

                _ ->
                    ( model, Cmd.none )

        RegistrationComplete _ ->
            ( model, Navigation.pushUrl model.key "/" )

        LoginEvent event ->
            case model.state of
                Login page ->
                    let
                        ( newPage, cmd ) =
                            updateLoginPage page event
                    in
                    ( setState model <| Login newPage, cmd )

                _ ->
                    ( model, Cmd.none )

        LoginComplete (Ok user) ->
            ( { model | user = Just user }
            , Cmd.batch
                [ Navigation.pushUrl model.key "/"
                , saveLoginData user
                ]
            )

        LoginComplete (Err _) ->
            ( model, Cmd.none )

        LinkClicked urlRequest ->
            case urlRequest of
                Browser.Internal url ->
                    ( model, Navigation.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Navigation.load href )

        UrlChanged url ->
            let
                ( state, cmd ) =
                    gotoRoute model.user <| Url.toString url
            in
            ( setState model state, cmd )

        Logout ->
            ( { model | user = Nothing }, Cmd.batch [ logout (), Navigation.pushUrl model.key "/" ] )


setState : Model -> ModelState -> Model
setState model state =
    { model | state = state }



-- VIEW


view : Model -> Browser.Document Msg
view model =
    let
        ui =
            case model.state of
                Register page ->
                    registrationView model.palette page

                Version ->
                    versionView model.palette

                Login page ->
                    loginView model.palette page

                NotFound ->
                    notFoundView model.palette

                Activity (Loaded (ViewCurrentActivity a)) ->
                    viewCurrentActivity model.palette a

                Activity (Loaded (NewActivityForm a)) ->
                    startActivityView model.palette a

                Activity (Loaded ActivityHomePage) ->
                    activityDashboardView model.palette

                Future _ ->
                    E.column [ E.spacing 30, contentWidth, E.centerX, E.centerY ]
                        [ header model.palette "Future"
                        , E.el [ E.centerX ] <| E.text "Coming Soon"
                        ]

                Past _ ->
                    E.column [ E.spacing 30, contentWidth, E.centerX, E.centerY ]
                        [ header model.palette "Past"
                        , E.el [ E.centerX ] <| E.text "Coming Soon (ironically)"
                        ]

                Settings ->
                    settingsView model.palette

                Activity Loading ->
                    E.text "Loading"

        buildIcon : String -> String -> (List (Icon.Attribute msg) -> E.Element Msg) -> E.Element Msg
        buildIcon label url icon =
            let
                name =
                    E.column [ E.centerX, E.spacing 5 ] [ E.el [ E.centerX ] <| icon [ Icon.width 35 ], E.text label ]
            in
            E.link [ E.width E.fill, E.height E.fill, E.centerX ] { url = url, label = name }

        nav =
            case model.user of
                Just _ ->
                    E.el [ Background.color model.palette.crust, E.width E.fill, E.height <| E.px 75 ] <|
                        E.row [ contentWidth, E.height E.fill, E.centerX, E.spaceEvenly ]
                            [ buildIcon "Past" "/past" Icons.caretLeftFilled
                            , buildIcon "Present" "/present" Icons.borderOutlined
                            , buildIcon "Future" "/future" Icons.caretRightFilled
                            , buildIcon "Settings" "/settings" Icons.settingOutlined
                            ]

                Nothing ->
                    E.none

        el =
            E.layout [ Background.color model.palette.base, E.width E.fill, E.height E.fill ] <|
                E.column
                    [ Font.color model.palette.text
                    , E.width <| E.px model.window.width
                    , E.height <| E.px model.window.height
                    , Background.color model.palette.base
                    ]
                    [ E.column [ E.centerX, E.spacing 15, E.paddingEach { top = 50, bottom = 0, left = 0, right = 0 } ]
                        [ E.image [ E.width <| E.px 75, E.height <| E.px 75, E.centerX ] { src = "/static/logo.webp", description = "Calming shapes" }
                        , E.el [ E.centerX, Font.color model.palette.primary, Font.size 25 ] <| E.text "Calm | Focused | Attentive"
                        ]
                    , E.el [ E.paddingXY 30 30, E.width E.fill, E.height E.fill, E.centerX, E.centerY ] ui
                    , nav
                    ]
    in
    { title = "Calm", body = [ el ] }


contentWidth : E.Attribute msg
contentWidth =
    E.width (E.fill |> E.maximum 600)



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ gotCredential (RegistrationEvent << GotCred)
        , gotLoginCredential (LoginEvent << GotLoginCred)
        , Time.every 60000 CheckToken
        ]


notFoundView : Colors.Palette -> E.Element msg
notFoundView _ =
    E.text "not found"



-- START ACTIVITY FORM


type MindPlatterActivityType
    = Sleep
    | Physical
    | Focus
    | Internal
    | Downtime
    | Playtime
    | Connecting


type alias StartActivityModel =
    { activityName : Maybe String
    , activityTypes : EverySet.EverySet MindPlatterActivityType
    , category : Maybe String
    , expectedDurationInMinutes : Maybe Int
    , startTime : Maybe Time.Posix
    , doNotDisturb : Bool
    }


emptyStartActivityForm : StartActivityModel
emptyStartActivityForm =
    { activityName = Nothing
    , activityTypes = EverySet.empty
    , category = Nothing
    , expectedDurationInMinutes = Nothing
    , startTime = Nothing
    , doNotDisturb = False
    }


type StartActivityMsg
    = SetName String
    | UnsetName
    | SetActivityType MindPlatterActivityType Bool
    | SetCategory String
    | SetExpectedTimeInMinutes (Maybe Int)
      -- | SetStartTime Time.Posix
    | SetDoNotDisturb Bool


startActivityUpdate : StartActivityModel -> StartActivityMsg -> StartActivityModel
startActivityUpdate model msg =
    case msg of
        SetName name ->
            { model | activityName = Just name }

        UnsetName ->
            { model | activityName = Nothing }

        SetActivityType activityType on ->
            if on then
                { model | activityTypes = EverySet.insert activityType model.activityTypes }

            else
                { model | activityTypes = EverySet.remove activityType model.activityTypes }

        SetCategory cat ->
            { model | category = Just cat }

        SetDoNotDisturb doNotDisturb ->
            { model | doNotDisturb = doNotDisturb }

        SetExpectedTimeInMinutes expectation ->
            { model | expectedDurationInMinutes = expectation }


startActivityView : Colors.Palette -> StartActivityModel -> E.Element Msg
startActivityView palette model =
    let
        buildIcon : MindPlatterActivityType -> E.Element Msg
        buildIcon t =
            let
                color =
                    if EverySet.member t model.activityTypes then
                        palette.primary

                    else
                        palette.text

                name =
                    E.column [ E.centerX, Font.color color, E.spacing 10 ] [ E.el [ E.centerX ] <| mindPlatterToIcon t [ Icon.width 50, Icon.fill color ], E.text (mindPlatterToString t) ]
            in
            Input.button [] { onPress = Just <| UpdateStartActivityForm <| SetActivityType t (not <| EverySet.member t model.activityTypes), label = name }
    in
    E.column [ E.spacing 30, E.width (E.fill |> E.maximum 600), E.centerX, E.centerY ]
        [ header palette "Start an Activity"
        , Input.text [ Font.color palette.base ] { onChange = UpdateStartActivityForm << SetName, text = Maybe.withDefault "" model.activityName, placeholder = Just (Input.placeholder [] <| E.text "Walk the dog"), label = Input.labelAbove [] <| E.text "Activity Name" }
        , E.column [ E.width E.fill, E.spacingXY 0 40 ]
            [ E.row [ E.width E.fill, E.spaceEvenly ]
                [ buildIcon Physical
                , buildIcon Playtime
                , buildIcon Focus
                , buildIcon Connecting
                ]
            , E.row [ E.width E.fill, E.spaceEvenly, E.paddingXY 25 0 ]
                [ buildIcon Internal
                , buildIcon Downtime
                , buildIcon Sleep
                ]
            ]
        , Input.text [ Font.color palette.base ] { onChange = UpdateStartActivityForm << SetCategory, text = Maybe.withDefault "" model.category, placeholder = Just (Input.placeholder [] <| E.text "exercise/swimming/intervals"), label = Input.labelAbove [] (E.text "Category") }
        , Input.checkbox [] { onChange = UpdateStartActivityForm << SetDoNotDisturb, icon = Input.defaultCheckbox, checked = model.doNotDisturb, label = Input.labelRight [] (E.text "Do Not Disturb Mode") }
        , Input.button [ E.paddingXY 20 15, Background.color palette.primary, Font.color palette.base, E.width E.fill ] { onPress = Just <| SubmitStartActivityForm model, label = E.el [ E.centerX ] <| E.text "Submit" }
        ]


activityDashboardView : Colors.Palette -> E.Element Msg
activityDashboardView palette =
    let
        buildIcon : { label : String, onPress : Msg, icon : List (Icon.Attribute msg) -> E.Element Msg } -> E.Element Msg
        buildIcon input =
            let
                name =
                    E.column [ E.centerX, E.spacing 10 ] [ E.el [ E.centerX ] <| input.icon [ Icon.width 50 ], E.text input.label ]
            in
            Input.button [ E.width E.fill, E.centerX ] { onPress = Just input.onPress, label = name }
    in
    E.column [ E.spacing 50, contentWidth, E.centerX, E.centerY ]
        [ header palette "Start an Activity"
        , E.column
            [ E.spacing 75, E.width E.fill ]
            [ E.row
                [ E.spaceEvenly, E.width E.fill ]
                [ buildIcon { onPress = StartActivity emptyStartActivityForm, label = "Manual Entry", icon = Icons.settingOutlined }
                , buildIcon { onPress = SubmitStartActivityForm { emptyStartActivityForm | activityName = Just "Sleep", activityTypes = EverySet.singleton Sleep }, label = "Sleep", icon = Icons.starOutlined }
                , buildIcon { onPress = SubmitStartActivityForm { emptyStartActivityForm | activityName = Just "Hygene", category = Just "hygene" }, label = "Hygene", icon = Icons.checkCircleOutlined }
                , buildIcon { onPress = StartActivity { emptyStartActivityForm | activityName = Just "Driving", category = Just "transit" }, label = "Driving", icon = Icons.carOutlined }
                ]
            , E.row
                [ E.spaceEvenly, E.width E.fill ]
                [ buildIcon { onPress = SubmitStartActivityForm { emptyStartActivityForm | activityName = Just "Family time", activityTypes = EverySet.singleton Connecting, category = Just "family_time" }, label = "Family", icon = Icons.teamOutlined }
                , buildIcon { onPress = StartActivity { emptyStartActivityForm | category = Just "house_work" }, label = "House", icon = Icons.homeOutlined }
                , buildIcon { onPress = StartActivity { emptyStartActivityForm | category = Just "work" }, label = "Work", icon = Icons.auditOutlined }
                , buildIcon { onPress = StartActivity { emptyStartActivityForm | activityName = Just "Meeting", category = Just "work/meeting", doNotDisturb = True }, label = "Meeting", icon = Icons.teamOutlined }
                ]
            , E.row
                [ E.spaceEvenly, E.width E.fill ]
                [ buildIcon { onPress = StartActivity { emptyStartActivityForm | activityName = Just "Coffee", category = Just "dining/coffee" }, label = "Coffee", icon = Icons.coffeeOutlined }
                , buildIcon { onPress = StartActivity { emptyStartActivityForm | category = Just "dining" }, label = "Dining", icon = Icons.appleOutlined }
                , buildIcon { onPress = SubmitStartActivityForm { emptyStartActivityForm | activityName = Just "Zwift", activityTypes = EverySet.singleton Physical, category = Just "exercise/cyclying/zwift" }, label = "Zwift", icon = Icons.pictureOutlined }
                , buildIcon { onPress = SubmitStartActivityForm { emptyStartActivityForm | activityName = Just "Running", activityTypes = EverySet.singleton Physical, category = Just "exercise/running" }, label = "Running", icon = Icons.heartOutlined }
                ]
            ]
        ]


mindPlatterToIcon : MindPlatterActivityType -> List (Icon.Attribute msg) -> E.Element msg
mindPlatterToIcon m =
    case m of
        Internal ->
            Icons.verticalAlignMiddleOutlined

        Connecting ->
            Icons.teamOutlined

        Playtime ->
            Icons.formatPainterOutlined

        Focus ->
            Icons.experimentOutlined

        Downtime ->
            Icons.youtubeOutlined

        Sleep ->
            Icons.starOutlined

        Physical ->
            Icons.settingOutlined


mindPlatterToString : MindPlatterActivityType -> String
mindPlatterToString m =
    case m of
        Internal ->
            "Time In"

        Connecting ->
            "Connecting"

        Playtime ->
            "Play Time"

        Focus ->
            "Focus"

        Downtime ->
            "Down Time"

        Sleep ->
            "Sleep"

        Physical ->
            "Physical"


submitStartActivityForm : StartActivityModel -> User -> Cmd Msg
submitStartActivityForm form user =
    let
        mindPlatterToJson : MindPlatterActivityType -> String
        mindPlatterToJson m =
            case m of
                Internal ->
                    "time_in"

                Connecting ->
                    "connecting_time"

                Playtime ->
                    "play_time"

                Focus ->
                    "focus_time"

                Downtime ->
                    "down_time"

                Sleep ->
                    "sleep"

                Physical ->
                    "physical_time"

        jsonVal =
            Encode.object
                [ ( "name", Encode.string <| Maybe.withDefault "" form.activityName )
                , ( "mind_platter_category", Encode.list Encode.string <| List.map mindPlatterToJson <| EverySet.toList form.activityTypes )
                , ( "categories", Encode.list Encode.string <| String.split "/" <| Maybe.withDefault "" form.category )

                -- , ("expected_duration_in_minutes", [])
                , ( "do_not_disturb", Encode.bool form.doNotDisturb )
                ]
    in
    Http.request
        { method = "POST"
        , url = "/activities"
        , body = Http.jsonBody jsonVal
        , expect = Http.expectJson GotStartActivityResponse Decode.string
        , headers = [ Http.header "Authorization" <| bearer user.accessToken ]
        , timeout = Just 1000
        , tracker = Nothing
        }


bearer : String -> String
bearer s =
    "Bearer " ++ s



-- ACTIVITY


type Loading a
    = Loading
    | Loaded a


type ActivityModel
    = ViewCurrentActivity CurrentActivity
    | NewActivityForm StartActivityModel
    | ActivityHomePage


type alias CurrentActivity =
    { name : String
    , mindPlatterActivityTypes : List MindPlatterActivityType
    , categories : List String
    }


currentActivityDecoder : Decode.Decoder (Maybe CurrentActivity)
currentActivityDecoder =
    let
        mindPlatterDecoder : String -> Decode.Decoder MindPlatterActivityType
        mindPlatterDecoder raw =
            case raw of
                "time_in" ->
                    Decode.succeed Internal

                "connecting_time" ->
                    Decode.succeed Connecting

                "play_time" ->
                    Decode.succeed Playtime

                "focus_time" ->
                    Decode.succeed Focus

                "down_time" ->
                    Decode.succeed Downtime

                "sleep" ->
                    Decode.succeed Sleep

                "physical_time" ->
                    Decode.succeed Physical

                _ ->
                    Decode.fail "unknown mind platter activity type"
    in
    Decode.maybe <|
        Decode.map3 CurrentActivity
            (Decode.field "name" Decode.string)
            (Decode.field "mind_platter_categories" <| Decode.map (Maybe.withDefault []) <| Decode.nullable <| Decode.list <| Decode.andThen mindPlatterDecoder Decode.string)
            (Decode.field "categories" <| Decode.map (Maybe.withDefault []) <| Decode.nullable <| Decode.list Decode.string)


fetchCurrentActivity : User -> Cmd Msg
fetchCurrentActivity user =
    Http.request
        { method = "GET"
        , url = "/activities/current"
        , expect = Http.expectJson GotCurrentActivity currentActivityDecoder
        , body = Http.emptyBody
        , headers = [ Http.header "Authorization" <| bearer user.accessToken ]
        , timeout = Just 1000
        , tracker = Nothing
        }


header : Colors.Palette -> String -> E.Element Msg
header palette title =
    E.el [ E.centerX, Font.color palette.blue, Font.size 25 ] <| E.text title


viewCurrentActivity : Colors.Palette -> CurrentActivity -> E.Element Msg
viewCurrentActivity palette act =
    E.column [ E.spacing 30, contentWidth, E.centerX, E.centerY ]
        [ header palette "Current Activity"
        , E.el [ E.centerX, Font.size 23 ] <| E.text act.name
        , E.row [ E.spacingXY 25 0, E.centerX ] <|
            List.map
                (\activity ->
                    E.column [ E.centerX, Font.color palette.blue, E.spacing 10 ]
                        [ E.el [ E.centerX ] <| mindPlatterToIcon activity [ Icon.width 25, Icon.fill palette.blue ]
                        , E.text <| mindPlatterToString activity
                        ]
                )
                act.mindPlatterActivityTypes
        , E.row [ E.spacingXY 5 0, E.centerX, Font.variant Font.smallCaps ] <| List.intersperse (Icons.rightOutlined []) <| List.map E.text act.categories
        , Input.button [ E.paddingXY 20 15, Background.color palette.primary, Font.color palette.base, E.centerX ] { onPress = Just EndActivity, label = E.text "End" }
        ]


endActivity : User -> Cmd Msg
endActivity user =
    Http.request
        { method = "POST"
        , url = "/activities/end"
        , body = Http.jsonBody <| Encode.object []
        , expect = Http.expectWhatever GotEndActivityResponse
        , headers = [ Http.header "Authorization" <| bearer user.accessToken ]
        , timeout = Just 1000
        , tracker = Nothing
        }



--- REGISTRATION


port requestRegistrationCredentials : Encode.Value -> Cmd msg


port gotCredential : (Encode.Value -> msg) -> Sub msg


type RegistrationPage
    = RequestingCredentialName String String
    | LoadingOptions String String
    | RequestingPasskey String String
    | RegistrationError Http.Error String String
    | LoadingCredentialResponse


registrationView : Colors.Palette -> RegistrationPage -> E.Element Msg
registrationView palette page =
    case page of
        RequestingCredentialName username _ ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Register"
                , Input.username [ Input.focusedOnLoad, Font.color palette.base ] { onChange = RegistrationEvent << UpdateUsername, text = username, placeholder = Just (Input.placeholder [] <| E.text "yubikey_mini"), label = Input.labelAbove [] <| E.text "Device Name" }
                , Input.button [ E.paddingXY 20 15, E.width E.fill, Background.color palette.primary, Font.color palette.base ] { onPress = Just <| RegistrationEvent SubmitUsername, label = E.el [ E.centerX ] <| E.text "Register Device" }
                ]

        LoadingOptions username _ ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Register"
                , Input.text [] { onChange = RegistrationEvent << UpdateUsername, text = username, placeholder = Just (Input.placeholder [] <| E.text "john_doe"), label = Input.labelAbove [] <| E.text "Username" }
                , Input.button [ E.paddingXY 20 15, Background.color palette.overlay2, Font.color palette.base ] { onPress = Nothing, label = E.text "Waiting for registration info" }
                ]

        RequestingPasskey username _ ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Register"
                , Input.text [] { onChange = RegistrationEvent << UpdateUsername, text = username, placeholder = Just (Input.placeholder [] <| E.text "john_doe"), label = Input.labelAbove [] <| E.text "Username" }
                , E.el [ E.paddingXY 20 15, Font.color palette.primary ] <| E.text "Waiting for authentication"
                ]

        RegistrationError _ _ _ ->
            E.column [ E.spacingXY 0 30 ]
                [ E.text "error"
                ]

        LoadingCredentialResponse ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Register"
                , E.text "Loading Credentials"
                ]


type RegistrationMsg
    = UpdateUsername String
    | SubmitUsername
    | GotRegistrationOptions (Result Http.Error Encode.Value)
    | GotCred Encode.Value


updateRegistrationPage : RegistrationPage -> RegistrationMsg -> ( RegistrationPage, Cmd Msg )
updateRegistrationPage page msg =
    case msg of
        UpdateUsername newName ->
            case page of
                RequestingCredentialName _ token ->
                    ( RequestingCredentialName newName token, Cmd.none )

                _ ->
                    ( page, Cmd.none )

        SubmitUsername ->
            case page of
                RequestingCredentialName username token ->
                    ( LoadingOptions username token, fetchRegistrationOptions username token )

                _ ->
                    ( page, Cmd.none )

        GotRegistrationOptions (Err err) ->
            case page of
                LoadingOptions username token ->
                    ( RegistrationError err username token, Cmd.none )

                _ ->
                    ( page, Cmd.none )

        GotRegistrationOptions (Ok val) ->
            case page of
                LoadingOptions username token ->
                    ( RequestingPasskey username token, requestRegistrationCredentials val )

                _ ->
                    ( page, Cmd.none )

        GotCred val ->
            ( LoadingCredentialResponse, sendRegistrationCredentials val )


fetchRegistrationOptions : String -> String -> Cmd Msg
fetchRegistrationOptions username token =
    Http.post
        { url = "/registration/begin/" ++ token
        , body = Http.jsonBody <| Encode.object [ ( "credential_name", Encode.string username ) ]
        , expect = Http.expectJson (RegistrationEvent << GotRegistrationOptions) Decode.value
        }


sendRegistrationCredentials : Encode.Value -> Cmd Msg
sendRegistrationCredentials val =
    Http.post
        { url = "/registration/end"
        , body = Http.jsonBody val
        , expect = Http.expectWhatever RegistrationComplete
        }



--- LOGIN


port requestLoginCredentials : Encode.Value -> Cmd msg


port gotLoginCredential : (Encode.Value -> msg) -> Sub msg


type LoginPage
    = ShowLoginButton
    | LoadingLoginOptions
    | RequestingLoginPasskey
    | LoginError Http.Error
    | LoadingLoginCredentialResponse


loginView : Colors.Palette -> LoginPage -> E.Element Msg
loginView palette page =
    case page of
        ShowLoginButton ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Login"
                , Input.button [ Input.focusedOnLoad, E.paddingXY 20 15, E.width E.fill, Background.color palette.primary, Font.color palette.base ] { onPress = Just <| LoginEvent SubmitLoginUsername, label = E.el [ E.centerX ] <| E.text "Begin Login" }
                ]

        LoadingLoginOptions ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Login"
                , E.text "Loading"
                ]

        RequestingLoginPasskey ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Login"
                , E.el [ Font.color palette.primary, E.centerX ] <| E.text "Waiting for authentication."
                ]

        LoginError _ ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Login"
                , E.text <| "error"
                ]

        LoadingLoginCredentialResponse ->
            E.column [ E.spacingXY 0 30, E.centerX, contentWidth, E.centerY ]
                [ header palette "Login"
                , E.text "Loading"
                ]


port saveLoginData : User -> Cmd msg


type LoginMsg
    = UpdateLoginUsername String
    | SubmitLoginUsername
    | GotLoginOptions (Result Http.Error Encode.Value)
    | GotLoginCred Encode.Value


updateLoginPage : LoginPage -> LoginMsg -> ( LoginPage, Cmd Msg )
updateLoginPage page msg =
    case msg of
        UpdateLoginUsername _ ->
            case page of
                ShowLoginButton ->
                    ( ShowLoginButton, Cmd.none )

                _ ->
                    ( page, Cmd.none )

        SubmitLoginUsername ->
            case page of
                ShowLoginButton ->
                    ( LoadingLoginOptions, fetchLoginOptions )

                _ ->
                    ( page, Cmd.none )

        GotLoginOptions (Err err) ->
            case page of
                LoadingLoginOptions ->
                    ( LoginError err, Cmd.none )

                _ ->
                    ( page, Cmd.none )

        GotLoginOptions (Ok val) ->
            case page of
                LoadingLoginOptions ->
                    ( RequestingLoginPasskey, requestLoginCredentials val )

                _ ->
                    ( page, Cmd.none )

        GotLoginCred val ->
            ( LoadingLoginCredentialResponse, sendLoginCredentials val )


fetchLoginOptions : Cmd Msg
fetchLoginOptions =
    Http.post
        { url = "/login/begin"
        , body = Http.emptyBody
        , expect = Http.expectJson (LoginEvent << GotLoginOptions) Decode.value
        }


sendLoginCredentials : Encode.Value -> Cmd Msg
sendLoginCredentials val =
    Http.post
        { url = "/login/end"
        , body = Http.jsonBody val
        , expect = Http.expectJson LoginComplete decodeUser
        }


type alias User =
    { accessToken : String
    , username : String
    , name : String
    , expires_at : Int
    }


decodeUser : Decode.Decoder User
decodeUser =
    Decode.map4 User
        (Decode.field "access_token" Decode.string)
        (Decode.field "username" Decode.string)
        (Decode.field "name" Decode.string)
        (Decode.field "expires_at" Decode.int)


port logout : () -> Cmd msg


settingsView : Colors.Palette -> E.Element Msg
settingsView palette =
    E.column [ E.spacing 30, contentWidth, E.centerX, E.centerY ]
        [ header palette "Settings"
        , E.column [ E.centerX, E.width E.fill ]
            [ E.link [ E.mouseOver [ Background.color palette.crust ], E.width E.fill, E.height <| E.px 50, E.paddingXY 20 0 ] { label = E.el [ E.centerY ] <| E.text "Version", url = "/version" }
            , Input.button [ E.mouseOver [ Background.color palette.crust ], E.width E.fill, E.height <| E.px 50, E.paddingXY 20 0 ] { label = E.text "Logout", onPress = Just Logout }
            ]
        ]


versionView : Colors.Palette -> E.Element Msg
versionView palette =
    E.column [ E.spacing 30, contentWidth, E.centerX, E.centerY ]
        [ header palette "Version"
        , E.el [ Font.color palette.primary, E.centerX ] <|
            E.text version
        , E.row
            [ E.centerX, E.spacing 30 ]
            [ Input.button
                [ E.centerX, Background.color palette.blue, Font.color palette.base, E.paddingXY 15 10 ]
                { onPress = Just <| ClearCache, label = E.text "Clear cache" }
            , E.link
                [ E.centerX, Border.color palette.blue, Border.width 1, Font.color palette.blue, E.paddingXY 15 10 ]
                { url = "/", label = E.text "Home" }
            ]
        ]
