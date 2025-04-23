build-app:
	cd app && elm make src/Activities.elm --output ../api/static/activities.js && cd --
