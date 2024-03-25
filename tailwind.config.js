/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/ui/**/*.templ}", "./**/*.templ",
    "./node_modules/flowbite/**/*.js"],
  theme: {
    extend: {},
  },
  plugins: [
    require('flowbite/plugin')
  ],
}

