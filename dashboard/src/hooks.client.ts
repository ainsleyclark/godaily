import { goto } from '$app/navigation';
import { auth } from '$lib/stores/auth';
import { toast } from 'svelte-sonner';

if (typeof window !== 'undefined') {
	window.addEventListener('metrics:unauthorized', () => {
		auth.clearSecret();
		toast.error('Session expired. Please sign in again.');
		void goto('/login');
	});
}
