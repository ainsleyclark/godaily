/**
 * subscribe.ts
 *
 * Handles the homepage hero subscribe form. Validates the email client-side,
 * POSTs to /api/subscribe, then shows an inline success or error state.
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

	input.classList.remove('hero__input--error');
	input.disabled = true;
	button.disabled = true;

	try {
		const res = await fetch('/api/subscribe', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email: value }),
		});

		if (res.ok) {
			button.textContent = "✓ You're subscribed!";
			button.classList.remove('button--primary');
			button.classList.add('button--success');
		} else if (res.status === 409) {
			setHint(hint, "You're already subscribed.");
			input.disabled = false;
			button.disabled = false;
		} else {
			setHint(hint, 'Something went wrong. Please try again.');
			input.disabled = false;
			button.disabled = false;
		}
	} catch {
		setHint(hint, 'Something went wrong. Please try again.');
		input.disabled = false;
		button.disabled = false;
	}
}

function setHint(el: HTMLElement | null, message: string): void {
	if (!el) return;
	el.textContent = message;
	el.hidden = false;
}
