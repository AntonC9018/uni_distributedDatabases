webapp := webapp

webapp_exe := webapp
ifeq ($(OS), Windows_NT)
    webapp_exe := $(webapp_exe).exe
endif

webapp_path := ./source/$(webapp)
node_project_dir := .

dev_webapp_output := ./tmp/$(webapp_exe)


.PHONY: build
build:
	# $(MAKE) tailwind-build
	$(MAKE) vite-build
	$(MAKE) templ-generate
	$(MAKE) go-build

.PHONY: go-build
go-build:
	go build -o $(dev_webapp_output) $(webapp_path)

# Needs to be run manually
.PHONY: init
init:
	go install github.com/cosmtrek/air@latest
	go install github.com/a-h/templ/cmd/templ@latest
	cd $(node_project_dir); npm install

.PHONY: run
run:
	$(MAKE) build
	$(dev_webapp_output)

.PHONY: build_prod
build_prod:
	rm -rf bin/*
	mkdir -p bin
	mkdir bin/dist
	# $(MAKE) tailwind-build
	$(MAKE) vite-build
	$(MAKE) templ-generate
	go build -ldflags "-X main.Environment=production" -o ./bin/$(webapp_exe) $(webapp_path)
	cp -r static/dist bin
	echo 'export APP_ENV=PRODUCTION; export GIN_MODE=release;./webapp'> bin/run.sh
	chmod +x bin/run.sh


css_dir := static/css/
dist_dir := static/dist/
dist_css_dir := $(dist_dir)css/
entry_css_file := $(css_dir)main.css
min_css_file := $(dist_css_dir)main.min.css

.PHONY: tailwind-watch
tailwind-watch:
	npx postcss $(entry_css_file) -o $(min_css_file) --watch

.PHONY: tailwind-build
tailwind-build:
	npx postcss $(entry_css_file) -o $(min_css_file)

.PHONY: vite-build
vite-build:
	npx tsc
	npx vite build

.PHONY: vite-watch
vite-watch:
	npx vite


.PHONY: templ-generate
templ-generate:
	templ generate -path $(webapp_path)

.PHONY: templ-watch
templ-watch:
	templ generate -path $(webapp_path) --watch

