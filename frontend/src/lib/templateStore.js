import {writable, derived} from 'svelte/store';

export const templates = writable([]);
export const insertTemplate = writable('');
export const defaultTemplateText = derived(templates, ($templates) => {
	const def = $templates.find((t) => t.is_default);
	return def ? def.text : '';
});
