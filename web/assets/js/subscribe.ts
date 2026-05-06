/**
 * subscribe.ts
 *
 * Handles the homepage hero subscribe form. V1 is UI-only: validate the email
 * client-side and swap the submit button to a fake success state. Wiring to a
 * real /subscribe endpoint will come in a follow-up.
 */

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function initSubscribeForm(): void {
	const forms = document.querySelectorAll<HTMLFormElement>('[data-subscribe]');
	forms.forEach((form) => form.addEventListener('submit', handleSubmit));
}

function handleSubmit(event: Event): void {
	event.preventDefault();
	const form = event.currentTarget as HTMLFormElement;
	const input = form.querySelector<HTMLInputElement>('input[type="email"]');
	const button = form.querySelector<HTMLButtonElement>('button[type="submit"]');
	if (!input || !button) return;

	const value = input.value.trim();
	if (!EMAIL_PATTERN.test(value)) {
		input.classList.add('hero__input--error');
		input.focus();
		return;
	}

	input.classList.remove('hero__input--error');
	input.disabled = true;
	button.disabled = true;
	button.textContent = "✓ You're on the list";
	button.classList.remove('button--primary');
	button.classList.add('button--success');
}
