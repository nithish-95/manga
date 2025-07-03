/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "/Users/nithish/Desktop/Codeing_Practice/Projects/manga/frontend/templates/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#6366F1',
          '50': '#E0E7FF',
          '100': '#C7D2FE',
          '200': '#A5B4FC',
          '300': '#818CF8',
          '400': '#6366F1',
          '500': '#4F46E5',
          '600': '#4338CA',
          '700': '#3730A3',
          '800': '#312E81',
          '900': '#282567',
        },
        secondary: '#EC4899',
        background: '#F9FAFB',
        card: '#FFFFFF',
        text: '#111827',
        'text-light': '#6B7280',
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
    },
  },
  plugins: [],
}