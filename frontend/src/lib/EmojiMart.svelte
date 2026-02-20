<script>
	import data from '@emoji-mart/data';
	import { Picker } from 'emoji-mart';
	import { onDestroy } from 'svelte';
	import { darkMode } from '$lib/settingsStore.js';

	let emojiPickerEl;
	let picker;

	let { select } = $props();

	// Wait for darkMode and language to be initialized before creating picker
	$effect(() => {
		if ($darkMode !== undefined && emojiPickerEl && !picker) {
			createPicker();
		}
	});

	// Update picker theme when darkMode changes
	$effect(() => {
		if (picker && $darkMode !== undefined) {
			picker.update({ theme: $darkMode ? 'dark' : 'light' });
		}
	});

	function createPicker() {
		picker = new Picker({
			data,
			theme: $darkMode ? 'dark' : 'light',
			autoFocus: true,
			onEmojiSelect: (emoji) => {
				select(emoji);
			}
		});
		emojiPickerEl.appendChild(picker);
	}

	onDestroy(() => {
		// the clickoutside handler is not unregistered properly, so this is probably redundant
		picker = null;
		emojiPickerEl = null;
	});
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore element_invalid_self_closing_tag -->
<div bind:this={emojiPickerEl} />
