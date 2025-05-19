const inputs: NodeListOf<HTMLInputElement> = document.querySelectorAll(
  ".digit-input",
);

inputs.forEach((input, index) => {
  input.addEventListener("input", (e) => {
    const target = e.target as HTMLInputElement;
    target.value = target.value.replace(/[^0-9]/g, "").slice(0, 1);

    if (target.value.length === 1) {
      const nextInput = inputs[index + 1];
      if (nextInput) {
        nextInput.focus();
      }
    }
  });

  input.addEventListener("keydown", (e) => {
    if (e.key === "Backspace" && input.value === "") {
      const prevInput = inputs[index - 1];
      if (prevInput) {
        prevInput.focus();
      }
    }
  });
});

const form: HTMLFormElement | null = document.querySelector("#form");
const otp: HTMLInputElement | null = document.querySelector("#code");

if (form !== null && otp !== null && inputs.length > 0) {
  form.addEventListener("submit", (event) => {
    event.preventDefault();

    const code = Array.from(inputs)
      .map((input) => input.value.trim())
      .join("");

    if (!/^\d+$/.test(code)) {
      alert("Please enter a valid numeric code.");
      return;
    }

    otp.value = code;
    form.submit(); // ✅ manually submit after modification
  });
}
