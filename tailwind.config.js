/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["internal/server/ui/**/*.templ"],
  // Can't have dynamic values via string interpolation
  // Need this safelist for gradients on comments to work
  safelist: Array.from(new Array(20), (_, i) => [`to-${i * 5}%`, `from-${i * 5}%`]).flat().concat(Array.from(new Array(101), (_, i) => `w-${i}%`)),
  theme: {
    fontFamily: {
      bebas: ['Bebas Neue', 'sans-serif'],
    },
    extend: {},
  },
  plugins: [],
}

