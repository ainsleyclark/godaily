export function initShareButtons(): void {
	document.querySelectorAll<HTMLButtonElement>('[data-share-copy]').forEach((btn) => {
		btn.addEventListener('click', () => handleCopy(btn));
	});
}

async function handleCopy(btn: HTMLButtonElement): Promise<void> {
	const shareUrl = btn.dataset.shareCopy ?? window.location.href;
	try {
		await navigator.clipboard.writeText(shareUrl);
	} catch {
		const ta = document.createElement('textarea');
		ta.value = shareUrl;
		ta.style.cssText = 'position:fixed;opacity:0;pointer-events:none';
		document.body.appendChild(ta);
		ta.select();
		document.execCommand('copy');
		document.body.removeChild(ta);
	}
	const label = btn.querySelector<HTMLElement>('[data-share-copy-label]');
	const original = label?.textContent ?? 'Copy link';
	if (label) label.textContent = 'Copied!';
	btn.classList.add('share-buttons__btn--copied');
	setTimeout(() => {
		if (label) label.textContent = original;
		btn.classList.remove('share-buttons__btn--copied');
	}, 2000);
}
