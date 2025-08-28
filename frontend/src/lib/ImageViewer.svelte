<script>
	import { onMount } from 'svelte';
	import { fade, slide } from 'svelte/transition';
	import { faXmark, faChevronRight, faChevronLeft } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	let { images } = $props(); // Array of image objects with `src`, `filename`, and `uuid_filename`

	let fullscreen = $state(false);
	let currentIndex = $state(0);

	function openFullscreen(index) {
		fullscreen = true;
		currentIndex = index;
	}

	function closeFullscreen() {
		fullscreen = false;
	}

	function nextImage() {
		if (currentIndex < images.length - 1) {
			currentIndex++;
		}
	}

	function prevImage() {
		if (currentIndex > 0) {
			currentIndex--;
		}
	}

	function handleKeyDown(event) {
		if (!fullscreen) return;

		if (event.key === 'ArrowRight') {
			nextImage();
		} else if (event.key === 'ArrowLeft') {
			prevImage();
		} else if (event.key === 'Escape') {
			closeFullscreen();
		}
	}

	// Variables for touch events
	let touchStartX = 0;
	let touchEndX = 0;

	// Swipe-Handler
	function handleTouchStart(event) {
		touchStartX = event.touches[0].clientX; // save the initial touch position
		touchEndX = touchStartX;
	}

	function handleTouchMove(event) {
		console.log('move');
		touchEndX = event.touches[0].clientX; // update the touch position
	}

	function handleTouchEnd(event) {
		console.log(event.target.classList);
		if (
			event.target.classList.contains('fullscreen-scroll') ||
			event.target.classList.contains('image')
		) {
			return; // do nothing if the touch ends on the scroll area
		}

		// calculate the swipe distance
		if (touchStartX - touchEndX > 50) {
			// Swipe left
			nextImage();
		} else if (touchEndX - touchStartX > 50) {
			// Swipe right
			prevImage();
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeyDown);
		return () => {
			window.removeEventListener('keydown', handleKeyDown);
		};
	});

	let fullscreenContainer = $state();
	$effect(() => {
		if (fullscreen && fullscreenContainer) {
			document.body.appendChild(fullscreenContainer);
		} else {
			fullscreenContainer?.remove();
		}
	});
</script>

<!-- Fullscreen Image Viewer -->
{#if fullscreen}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		bind:this={fullscreenContainer}
		transition:fade={{ duration: 100 }}
		class="fullscreen-overlay"
		onclick={(event) => {
			if (
				event.target.classList.contains('fullscreen-image-container') ||
				event.target.classList.contains('fullscreen-overlay')
			) {
				closeFullscreen();
			}
		}}
		ontouchstart={handleTouchStart}
		ontouchmove={handleTouchMove}
		ontouchend={handleTouchEnd}
	>
		<button class="close-btn" onclick={closeFullscreen}><Fa icon={faXmark} fw /> </button>
		<div class="fullscreen-image-container">
			<button class="nav-btn prev" onclick={prevImage} disabled={currentIndex === 0}>
				<Fa icon={faChevronLeft} fw />
			</button>
			{#key images[currentIndex].uuid_filename}
				<img
					class="fullscreen-image"
					alt={images[currentIndex].filename}
					src={images[currentIndex].src}
				/>
			{/key}
			<div class="image-info">
				<span class="image-name">{images[currentIndex].filename}</span>
				<a
					class="download-btn"
					href={images[currentIndex].src}
					download={images[currentIndex].filename}
				>
					{$t('imageViewer.download')}
				</a>
			</div>
			<button
				class="nav-btn next"
				onclick={nextImage}
				disabled={currentIndex === images.length - 1}
			>
				<Fa icon={faChevronRight} fw />
			</button>
		</div>
		<div class="horizontal-scroll fullscreen-scroll">
			{#each images as image, index (image.uuid_filename)}
				<button
					type="button"
					class="image-container {index === currentIndex ? 'active' : ''}"
					onclick={() => (currentIndex = index)}
				>
					<img class="image" alt={image.filename} src={image.src} />
				</button>
			{/each}
		</div>
	</div>
{/if}

<!-- <div class="image-gallery"> -->
<div class="horizontal-scroll px-2">
	{#each images as image, index (image.uuid_filename)}
		<button
			type="button"
			class="image-container"
			onclick={() => openFullscreen(index)}
			transition:slide={{ axis: 'x' }}
		>
			<img class="image" alt={image.filename} src={image.src} transition:fade />
		</button>
	{/each}
</div>

<style>
	.image-info {
		position: absolute;
		bottom: 0;
		right: 0;
		background: rgba(0, 0, 0, 0); /* Semi-transparent background */
		color: white;
		display: flex;
		justify-content: right;
		align-items: center;
		padding: 0.5rem 1rem;
		box-sizing: border-box;
		transition: background 0.2s ease;
	}

	.image-info:hover {
		background: rgba(0, 0, 0, 0.6);
	}

	.image-name {
		font-size: 1rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		padding-right: 1rem;
	}

	.download-btn {
		color: white;
		text-decoration: none;
		font-size: 1rem;
		background: rgba(255, 255, 255, 0.2);
		padding: 0.3rem 0.6rem;
		border-radius: 4px;
		transition: background 0.3s ease;
	}

	.download-btn:hover {
		background: rgba(255, 255, 255, 0.4);
	}

	.horizontal-scroll {
		display: flex;
		gap: 1rem;
		overflow-x: auto;
		padding: 0.5rem 0;
	}

	.image-container {
		border: none;
		background: transparent;
		padding: 0;
		cursor: pointer;
		transition: transform 0.2s ease;
	}

	.image-container:hover .image {
		transform: scale(1.1);
		box-shadow: 0 0 12px 3px rgba(0, 0, 0, 0.2);
	}

	.image {
		max-width: 150px;
		max-height: 100px;
		border-radius: 8px;
		transition: transform 0.3s ease;
	}

	.fullscreen-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		background: rgba(0, 0, 0, 0.9);
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		align-items: center;
		z-index: 9999;
		color: white;
		pointer-events: all;
	}

	.fullscreen-image-container {
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		width: 100%;
		height: 80%;
		flex: 1 0;
	}

	.fullscreen-image {
		max-width: 100%;
		max-height: 95%;
		border-radius: 8px;
	}

	.nav-btn {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		background: rgba(0, 0, 0, 0.5);
		border: none;
		color: white;
		font-size: 1.5rem;
		padding: 0.5rem 1rem;
		cursor: pointer;
		z-index: 10000;
	}

	.nav-btn.prev {
		left: 1rem;
	}

	.nav-btn.next {
		right: 1rem;
	}

	.nav-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.close-btn {
		display: flex;
		align-items: center;
		padding: 10px;

		position: absolute;
		top: 1rem;
		right: 1rem;
		background: none;
		border: none;
		color: white;
		font-size: 2rem;
		cursor: pointer;
		z-index: 10000;
	}

	.fullscreen-scroll {
		width: 100%;
		display: flex;
		justify-content: center;
		padding: 0.5rem;
		background: rgba(0, 0, 0, 0.8);
	}

	.fullscreen-scroll .image-container.active .image {
		outline: 2px solid white;
	}
</style>
