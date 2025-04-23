#!/usr/bin/env sh

cd api
go tool air &
  
cd ../app
js=../api/static/views/activities.js  
mjs=../api/static/views/activities.min.js  

elm make --output=$js src/Activities.elm 
cp $js $mjs

inotifywait -qmr -e modify ./src |
while read -r filename event; do
  elm make --output=$js src/Activities.elm 
  cp $js $mjs
done
cd ..
