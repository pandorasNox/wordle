/** @type {import('tailwindcss').Config} */
module.exports = {
    // content: [],
    // content: ["/css/**/*.{html,js}"],
    // content: ["/templates/*"],
    content: ["templates/**/*.{html,tmpl}"],
    safelist: [
        // 'bg-red-500',
        // 'text-3xl',
        // 'lg:text-4xl',
        // {
        //   pattern: /([a-zA-Z]+)-./, // all of tailwind
        // },
    ],
    darkMode: 'class',
    theme: {
        extend: {},
    },
    plugins: [],
}
