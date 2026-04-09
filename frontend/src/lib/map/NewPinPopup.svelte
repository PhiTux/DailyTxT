<script>
	import { onMount } from 'svelte';

	let { initialValue = '', onSave = () => {}, onChange = () => {} } = $props();

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
	<input
		id="newPinTextInput"
		type="text"
		class="form-control"
		placeholder="Name"
		bind:value={initialValue}
		oninput={handleInput}
		onkeydown={handleKeydown}
	/>
	<button type="button" class="btn btn-success" onclick={handleSave}>Speichern</button>
</div>
