module Tests.StartActivity exposing (..)

import Test exposing (Test, describe, test)
import Expect
import Activities.StartActivity exposing (..)
import EverySet 
suite : Test
suite =
    describe "start activity test"
        [ describe "setting the activity name"
            [ test "adding a name" <|
                \_ ->
                    let
                        name = "My new activity"
                        new = update empty <| SetName name
                    in
                        Expect.equal new.activityName (Just name)
            , test "unsetting the name" <|
                \_ ->
                    let
                        name = "My new activity"
                        new = update empty <| SetName name
                        next = update new <| UnsetName
                    in
                        Expect.equal next.activityName Nothing                
            ]
        , describe "setting activity types" 
            [ test "adding a new activity type" <|
                \_ ->
                    let
                        new = update empty <| AddActivityType Internal
                    in
                        Expect.equal True <| EverySet.member Internal new.activityTypes
            , test "removing an activity type" <|
                \_ ->
                    let
                        new = update empty <| AddActivityType Internal
                        next = update new <| RemoveActivityType Internal
                    in
                        Expect.equal False <| EverySet.member Internal next.activityTypes
            ]

        , describe "setting the category"
            [ test "setting the activity category" <|
                \_ ->
                    let
                        cat = "Some Category"
                        new = update empty <| SetCategory cat
                    in
                        Expect.equal (Just cat) new.category

            ]
        , describe "setting the do not disturb flag"
            [ test "setting do not disturb" <|
                \_ ->
                    let
                        new = update empty <| SetDoNotDisturb True
                    in
                        Expect.equal True new.doNotDisturb

            ]
            

        , describe "set the expected activity duration"
            [ test "set the expected duration" <|
                \_ ->
                    let
                        new = update empty <| SetExpectedTimeInMinutes <| Just 12
                    in
                        Expect.equal (Just 12) new.expectedDurationInMinutes

            ]
            
            
        ]
