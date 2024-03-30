/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["../**/*.templ",
    "./node_modules/flowbite/**/*.js"],
  // Can't have dynamic values via string interpolation
  // Need this safelist for gradients on comments to work
  safelist: [
    'to-5%',
    'to-10%',
    'to-15%',
    'to-20%',
    'to-25%',
    'to-30%',
    'to-35%',
    'to-40%',
    'to-45%',
    'to-50%',
    'to-55%',
    'to-60%',
    'to-65%',
    'to-70%',
    'to-75%',
    'to-80%',
    'to-85%',
    'to-90%',
    'to-95%',
    'from-5%',
    'from-10%',
    'from-15%',
    'from-20%',
    'from-25%',
    'from-30%',
    'from-35%',
    'from-40%',
    'from-45%',
    'from-50%',
    'from-55%',
    'from-60%',
    'from-65%',
    'from-70%',
    'from-75%',
    'from-80%',
    'from-85%',
    'from-90%',
    'from-95%',
  ],
  theme: {
    extend: {},
  },
  plugins: [
    require('flowbite/plugin')
  ],
}

