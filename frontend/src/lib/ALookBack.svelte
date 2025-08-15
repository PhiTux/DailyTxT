<script>
	import { marked } from 'marked';
	import { selectedDate } from './calendarStore';

	marked.use({
		breaks: true,
		gfm: true
	});

	let { log } = $props();

	let preview;
	let content;
	let modal;
	let isModalOpen = $state(false);

	function openModal() {
		if (!preview || !modal || !content) return;

		const previewRect = preview.getBoundingClientRect();
		const targetWidth = Math.min(window.innerWidth * 0.8, 600); // Target width

		// Initial state for the animation
		// Position and scale to match the button
		content.style.left = `${previewRect.left}px`;
		content.style.top = `${previewRect.top}px`;
		content.style.width = `${previewRect.width}px`;
		content.style.height = `${previewRect.height}px`;
		content.style.transform = 'scale(1)'; // Start at button's scale
		content.style.opacity = '0';

		modal.style.display = 'flex';

		void content.offsetWidth;

		// Target state for the animation
		// Calculate scale factor to reach targetWidth from previewRect.width
		const scaleX = targetWidth / previewRect.width;

		const targetLeft = (window.innerWidth - targetWidth) / 2;
		const targetTop = window.innerHeight * 0.2;

		content.style.left = `${targetLeft}px`; // Position for final state
		content.style.top = `${targetTop}px`; // Position for final state
		content.style.width = `${targetWidth}px`;
		// Height will be 'auto' or controlled by max-height in CSS
		content.style.height = 'auto'; // Let CSS max-height handle this
		content.style.transform = 'scale(1)'; // End at normal scale, but at new position/size
		content.style.opacity = '1';

		isModalOpen = true;
		document.body.style.overflow = 'hidden';
	}

	function closeModal() {
		if (!preview || !modal || !content) return;

		const previewRect = preview.getBoundingClientRect();

		content.style.left = `${previewRect.left}px`;
		content.style.top = `${previewRect.top}px`;
		content.style.width = `${previewRect.width}px`;
		content.style.height = `${previewRect.height}px`;
		content.style.transform = 'scale(1)';
		content.style.opacity = '0';

		setTimeout(() => {
			if (!isModalOpen) {
				modal.style.display = 'none';
				document.body.style.overflow = '';
			}
		}, 400);

		isModalOpen = false;
	}

	function handleKeydown(event) {
		if (event.key === 'Escape' && isModalOpen) {
			closeModal();
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<svelte:window on:keydown={handleKeydown} />

<!-- svelte-ignore a11y_consider_explicit_label -->
<button
	bind:this={preview}
	onclick={openModal}
	id="zoomButton"
	class="btn"
	style="width: 200px; height: 100px;"
>
	<div class="d-flex flex-row h-100">
		<div class="left d-flex flex-column justify-content-evenly px-1">
			<div><b>{log?.year}</b></div>
			<div><em><b>{log?.years_old}</b> J</em></div>
		</div>
		<div class="html-preview p-1">
			{@html marked.parse(log?.text)}
		</div>
	</div>
</button>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<div
	bind:this={modal}
	id="zoomModal"
	class="zoom-modal"
	onclick={(event) => {
		if (event.target === modal) {
			closeModal();
		}
	}}
>
	<div bind:this={content} class="zoom-content">
		<div class="zoom-content-header">
			<span
				>Vor {log?.years_old} Jahren | {new Date(
					log?.year,
					log?.month - 1,
					log?.day
				).toLocaleDateString('locale', {
					day: '2-digit',
					month: '2-digit',
					year: 'numeric'
				})}</span
			>
			<button
				type="button"
				class="btn-close btn-close-white"
				aria-label="Close"
				onclick={closeModal}
			></button>
		</div>
		<div class="modal-text">{@html marked.parse(log?.text)}</div>
		<button
			onclick={() => {
				$selectedDate = { year: log.year, month: log.month, day: log.day };
				closeModal();
			}}
			class="btn btn-primary"
			id="closeZoom">Öffnen</button
		>
	</div>
</div>

<style>
	.left {
		background-color: rgba(180, 180, 180, 0.45);
		border-top-left-radius: 0.375rem;
		border-bottom-left-radius: 0.375rem;
	}

	.modal-text {
		margin-left: 20px;
		margin-right: 20px;
		margin-top: 20px;
	}

	#closeZoom {
		margin-left: 20px;
		margin-bottom: 20px;
	}

	.zoom-content-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 8px 15px;
		background-color: #343a40;
		color: white;
		border-bottom: 1px solid #495057;
		flex-shrink: 0;
		border-top-left-radius: 8px;
		border-top-right-radius: 8px;
	}

	.html-preview :global(h1),
	.html-preview :global(h2),
	.html-preview :global(h3),
	.html-preview :global(h4),
	.html-preview :global(h5),
	.html-preview :global(h6) {
		font-size: 1.1em !important;
	}

	.html-preview {
		overflow: hidden;
		flex-grow: 1;
		max-height: 100%;
		line-height: 1.25;
		backdrop-filter: blur(8px) saturate(150%);
	}

	.html-preview::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		height: 40px;
		background: linear-gradient(to bottom, transparent, rgba(219, 219, 219, 0.45) 80%);
		pointer-events: none;
	}

	#zoomButton {
		background-color: rgba(219, 219, 219, 0.45);
		transition: 0.3s ease;
		padding: 0 !important;
	}

	#zoomButton:hover {
		background-color: rgba(219, 219, 219, 0.65);
	}

	.zoom-modal {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		display: none;
		justify-content: center;
		align-items: center;
		z-index: 1050;
		background-color: rgba(0, 0, 0, 0.5);
		overflow: hidden;
	}

	.zoom-content {
		position: absolute;
		background: white;
		color: black;
		border-radius: 8px;
		box-shadow: 0 0 20px rgba(0, 0, 0, 0.4);
		/* transform-origin: top left; */ /* Let's try center for smoother scaling if we use scale for size */
		transition:
			left 0.4s cubic-bezier(0.25, 0.8, 0.25, 1),
			top 0.4s cubic-bezier(0.25, 0.8, 0.25, 1),
			width 0.4s cubic-bezier(0.25, 0.8, 0.25, 1),
			height 0.4s cubic-bezier(0.25, 0.8, 0.25, 1),
			/* Animating height can be jittery */ transform 0.4s cubic-bezier(0.25, 0.8, 0.25, 1),
			opacity 0.4s ease-out;
		/* padding: 20px; */
		overflow-y: auto;
		max-width: 600px; /* Set max-width here */
		max-height: 80vh; /* Set max-height here */
		/* Consider adding will-change for properties you animate, but use sparingly */
		/* will-change: transform, opacity, left, top, width, height; */
	}
</style>
