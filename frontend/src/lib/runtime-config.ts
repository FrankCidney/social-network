const DEFAULT_LOCAL_API_URL = 'http://localhost:8080';

function normalizeUrl(url: string) {
	return url.replace(/\/$/, '');
}

export const API_BASE_URL = normalizeUrl(process.env.NEXT_PUBLIC_API_URL || DEFAULT_LOCAL_API_URL);

export function getWebSocketUrl() {
	return `${API_BASE_URL.replace(/^http/i, 'ws')}/api/ws`;
}
