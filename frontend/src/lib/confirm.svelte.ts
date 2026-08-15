export type ConfirmOptions = {
	title?: string;
	message: string;
	confirmLabel?: string;
	cancelLabel?: string;
	color?: 'primary' | 'red';
};

type ConfirmState = {
	open: boolean;
	title: string;
	message: string;
	confirmLabel: string;
	cancelLabel: string;
	color: 'primary' | 'red';
	resolve: ((value: boolean) => void) | null;
};

export const confirmState = $state<ConfirmState>({
	open: false,
	title: 'Konfirmasi',
	message: '',
	confirmLabel: 'Ya',
	cancelLabel: 'Batal',
	color: 'primary',
	resolve: null
});

export function confirm(options: ConfirmOptions): Promise<boolean> {
	return new Promise((resolve) => {
		confirmState.open = true;
		confirmState.title = options.title ?? 'Konfirmasi';
		confirmState.message = options.message;
		confirmState.confirmLabel = options.confirmLabel ?? 'Ya';
		confirmState.cancelLabel = options.cancelLabel ?? 'Batal';
		confirmState.color = options.color ?? 'primary';
		confirmState.resolve = resolve;
	});
}

export function resolveConfirm(value: boolean) {
	confirmState.resolve?.(value);
	confirmState.resolve = null;
	confirmState.open = false;
}
