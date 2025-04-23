#!/usr/bin/env sh

cd ./app

js=../api/static/views/activities.js  
min=../api/static/views/activities.min.js  

elm make --output=$js src/Activities.elm --optimize
uglifyjs $js --compress 'pure_funcs=[F2,F3,F4,F5,F6,F7,F8,F9,A2,A3,A4,A5,A6,A7,A8,A9],pure_getters,keep_fargs=false,unsafe_comps,unsafe' | uglifyjs --mangle --output $min
sed -i 's/slider-vertical//g' $min

echo "Compiled size:$(wc $js -c) bytes  ($js)"
echo "Minified size:$(wc $min -c) bytes  ($min)"
echo "Gzipped size: $(gzip $min -c | wc -c) bytes"

cd ../api

go build -o ../calm -tags "prod" .

cd ..
