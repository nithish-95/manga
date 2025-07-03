/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#3B82F6', // A vibrant blue
        secondary: '#6B7280', // A neutral gray
        accent: '#EC4899', // A bright pink for highlights
        background: '#F3F4F6', // Light gray for backgrounds
        card: '#FFFFFF', // White for cards and containers
        text: '#1F2937', // Dark gray for text
        'text-light': '#4B5563', // Lighter gray for secondary text
      },
    },
  },
  plugins: [],
}