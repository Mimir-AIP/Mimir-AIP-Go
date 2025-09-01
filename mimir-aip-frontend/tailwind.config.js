module.exports = {
  darkMode: 'class',
  content: [
    './src/**/*.{js,ts,jsx,tsx}',
    './components.json',
  ],
  theme: {
    extend: {
      colors: {
        navy: '#0B192C',
        blue: '#1E3E62',
        orange: '#FF6500',
        white: '#FFFFFF',
      },
      backgroundColor: {
        'navy-dark': '#0B192C',
      },
    },
  },
  plugins: [],
}
