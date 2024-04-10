// IIFE (Immediately Invoked Function Expression) / Self-Executing Anonymous Function
(function () {

    console.log("main.ts was executed")

    type State = {
        letters: Array<string>
    }

    function main(): void {
        initalThemeHandler();
        themeButtonToggleHandler();

        document.addEventListener('DOMContentLoaded', initKeyListener, false);
    }

    function initalThemeHandler() {
        // On page load or when changing themes, best to add inline in `head` to avoid FOUC
        if (localStorage.getItem('color-theme') === 'dark' || (!('color-theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark')
        }
    }

    function themeButtonToggleHandler(): void {
        const themeToggleDarkIcon = document.getElementById('theme-toggle-dark-icon');
        const themeToggleLightIcon = document.getElementById('theme-toggle-light-icon');
        if (themeToggleDarkIcon === null || themeToggleLightIcon === null) {
            return
        }

        // Change the icons inside the button based on previous settings
        if (localStorage.getItem('color-theme') === 'dark' || (!('color-theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            themeToggleLightIcon.classList.remove('hidden');
        } else {
            themeToggleDarkIcon.classList.remove('hidden');
        }

        const themeToggleBtn = document.getElementById('theme-toggle');
        if (themeToggleBtn === null) {
            return
        }

        themeToggleBtn.addEventListener('click', function() {

            // toggle icons inside button
            themeToggleDarkIcon.classList.toggle('hidden');
            themeToggleLightIcon.classList.toggle('hidden');

            // if set via local storage previously
            if (localStorage.getItem('color-theme')) {
                if (localStorage.getItem('color-theme') === 'light') {
                    document.documentElement.classList.add('dark');
                    localStorage.setItem('color-theme', 'dark');
                } else {
                    document.documentElement.classList.remove('dark');
                    localStorage.setItem('color-theme', 'light');
                }

            // if NOT set via local storage previously
            } else {
                if (document.documentElement.classList.contains('dark')) {
                    document.documentElement.classList.remove('dark');
                    localStorage.setItem('color-theme', 'light');
                } else {
                    document.documentElement.classList.add('dark');
                    localStorage.setItem('color-theme', 'dark');
                }
            }

        });

    }

    function initKeyListener(): void {
        let state: State = {
            letters: [],
        };
    
        const inputs: NodeListOf<HTMLInputElement> =
            document.querySelectorAll(".focusable");
    
        document.addEventListener('keyup', (e: KeyboardEvent) => {
            console.log("keyup event, code:", e.code, " / e.key", e.key, " / e.key.charCodeAt(0)", e.key.charCodeAt(0));

            const isSingleKey = e.key.length === 1
            const isInAllowedKeyRange = (65 <= e.key.charCodeAt(0) && e.key.charCodeAt(0) <= 122)
            const inputRowIsFillable = state.letters.length < inputs.length
            if (isSingleKey && isInAllowedKeyRange && inputRowIsFillable) {
                state.letters.push(e.key);
                updateInput(state, inputs);
            }

            if (e.key === "Backspace") {
                state.letters.pop();
                updateInput(state, inputs);
            }
    
            // console.log("nodes: ", inputs);
        });

        //TODO: listen htmx.AfterSwap... + uptate state etc....
    }

    function updateInput(state: State, inputs: NodeListOf<HTMLInputElement>): void {
        inputs.forEach((input: HTMLInputElement, index: number) => {
            inputs[index].value = state.letters[index] ?? '';

            if (state.letters.length === 0) {
                inputs[0].focus()
            } else if (state.letters.length === inputs.length) {
                inputs[state.letters.length-1].focus();
            } else if (state.letters.length <= inputs.length) {
                inputs[state.letters.length].focus();
            }
        });
    }

    main();

})();
