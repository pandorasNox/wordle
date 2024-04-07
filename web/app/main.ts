// IIFE (Immediately Invoked Function Expression) / Self-Executing Anonymous Function
(function () {

    console.log("main.ts was executed")

    type State = {
        letters: Array<string>
    }
    
    const ALLOWED_KEY_CODES = []
    
    function initListener(): void {
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

    document.addEventListener('DOMContentLoaded', initListener, false);

})();
