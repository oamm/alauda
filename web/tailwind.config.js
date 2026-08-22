/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          950: "#06111f",
          900: "#071827",
          850: "#0a2033",
          800: "#0f2a41",
        },
        alauda: {
          cyan: "#19d9e6",
          blue: "#1378ff",
          green: "#19d98a",
        },
      },
      boxShadow: {
        glow: "0 24px 80px rgb(0 217 230 / 18%)",
        panel: "0 18px 50px rgb(3 10 24 / 18%)",
      },
    },
  },
  plugins: [],
};
