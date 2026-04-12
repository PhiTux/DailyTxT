<script>
	import { onMount } from 'svelte';

	let {
		initialValue = '',
		onSave = () => {},
		onChange = () => {},
		fullScreen = false,
		translate = (key) => key
	} = $props();

	function tr(key) {
		return typeof translate === 'function' ? translate(key) : key;
	}

	onMount(() => {
		document.querySelector('#newPinTextInput')?.focus();
	});

	function handleInput(event) {
		initialValue = event.target.value;
		onChange(initialValue);
	}

	function handleSave(event) {
		event.preventDefault();
		event.stopPropagation();
		onSave(initialValue);
	}

	function handleKeydown(event) {
		if (event.key === 'Enter') {
			handleSave(event);
		}
	}
</script>

<div class="new-pin-popup">
	{#if fullScreen}
		{initialValue}
	{:else}
		<input
			id="newPinTextInput"
			type="text"
			class="form-control"
			placeholder={tr('modal.tag.name')}
			bind:value={initialValue}
			oninput={handleInput}
			onkeydown={handleKeydown}
		/>
		<button type="button" class="btn btn-success" onclick={handleSave}>{tr('settings.save')}</button
		>
	{/if}
</div>
