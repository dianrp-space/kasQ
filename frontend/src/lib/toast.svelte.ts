export type ToastColor = 'green' | 'red' | 'blue' | 'yellow';

export type ToastItem = {
	id: string;
	message: string;
	color: ToastColor;
};

export const toastState = $state({ items: [] as ToastItem[] });

function push(message: string, color: ToastColor) {
	const id = crypto.randomUUID();
	toastState.items = [...toastState.items, { id, message, color }];
	window.setTimeout(() => dismiss(id), 4500);
}

export function dismiss(id: string) {
	toastState.items = toastState.items.filter((t) => t.id !== id);
}

export const toast = {
	success: (message: string) => push(message, 'green'),
	error: (message: string) => push(message, 'red'),
	info: (message: string) => push(message, 'blue'),
	warn: (message: string) => push(message, 'yellow')
};
