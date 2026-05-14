/**
 * subscribe.ts
 *
 * Handles the homepage hero subscribe form. Validates the email client-side,
 * POSTs to /api/subscribe, then redirects to /thank-you/ on success or shows
 * an inline error state.
 */

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function initSubscribeForm(): void {
	const forms = document.querySelectorAll<HTMLFormElement>('[data-subscribe]');
	forms.forEach((form) => form.addEventListener('submit', handleSubmit));
}

async function handleSubmit(event: Event): Promise<void> {
	event.preventDefault();
	const form = event.currentTarget as HTMLFormElement;
	const input = form.querySelector<HTMLInputElement>('input[type="email"]');
	const button = form.querySelector<HTMLButtonElement>('button[type="submit"]');
	const hint = form.querySelector<HTMLElement>('[data-subscribe-hint]');
	if (!input || !button) return;

	const value = input.value.trim();
	if (!EMAIL_PATTERN.test(value)) {
		input.classList.add('hero__input--error');
		input.focus();
		return;
	}

	const originalText = button.textContent;
	input.classList.remove('hero__input--error');
	input.disabled = true;
	button.disabled = true;
	button.textContent = 'Subscribing...';
	button.classList.add('button--loading');

	try {
		const res = await fetch('/api/subscribe', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email: value }),
		});

		if (res.ok) {
			window.location.href = '/thank-you/';
		} else if (res.status === 409) {
			resetButton(button, hint, originalText, "You're already subscribed.");
			input.disabled = false;
		} else {
			resetButton(button, hint, originalText, 'Something went wrong. Please try again.');
			input.disabled = false;
		}
	} catch {
		resetButton(button, hint, originalText, 'Something went wrong. Please try again.');
		input.disabled = false;
	}
}

function resetButton(button: HTMLButtonElement, hint: HTMLElement | null, originalText: string | null, message: string): void {
	button.textContent = originalText;
	button.classList.remove('button--loading');
	button.disabled = false;
	setHint(hint, message);
}

function setHint(el: HTMLElement | null, message: string, isError = true): void {
	if (!el) return;
	el.textContent = message;
	el.hidden = false;
	el.classList.toggle('hero__hint--error', isError);
}
