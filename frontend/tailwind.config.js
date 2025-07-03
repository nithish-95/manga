module.exports = {
  content: ["/Users/nithish/Desktop/Codeing_Practice/Projects/manga/frontend/templates/**/*.html"],
  theme: {
    extend: {
      colors: {
        primary: '#6366f1',    // Vibrant indigo
        secondary: '#ec4899',  // Energetic pink
        accent: '#f59e0b',     // Golden yellow
        background: '#0f172a', // Deep navy
        card: '#1e293b',       // Slightly lighter navy
        surface: '#334155',    // Card hover state
        text: {
          primary: '#f1f5f9',  // Light gray
          secondary: '#cbd5e1', // Medium gray
        }
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        display: ['Poppins', 'sans-serif']
      }
    }
  },
  plugins: [],
}