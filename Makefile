.PHONY: templ
templ:
	templ generate --watch --proxy http://localhost:8080

serve:
	air

tailwind:
	npx tailwindcss -c ./tailwind.config.js -i ./public/styles.css -o ./public/output.css --watch
