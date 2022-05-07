.PHONY:
build: dist/output.css dist/index.html dist/index.js

dist/output.css: src/input.css
	mkdir -p dist
	npx tailwindcss -i $< -o $@ --minify

dist/%: src/%
	mkdir -p dist
	cp $< $@

.PHONY: serve-css
serve-css:
	npx tailwindcss -i src/input.css -o src/output.css --watch

.PHONY: serve-html
serve-html:
	cd src && python -m http.server
