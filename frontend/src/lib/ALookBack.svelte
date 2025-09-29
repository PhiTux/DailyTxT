<script>
	import { marked } from 'marked';
	import { selectedDate } from './calendarStore';
	import { getTranslate, getTolgee } from '@tolgee/svelte';
	import { onMount } from 'svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	marked.use({
		breaks: true,
		gfm: true
	});

	let { log } = $props();

	let modal;
	let modalInstance;

	onMount(() => {
		// Import Bootstrap Modal
		import('bootstrap').then((bootstrap) => {
			modalInstance = new bootstrap.Modal(modal, {
				backdrop: true,
				keyboard: true
			});
		});
	});

	function openModal() {
		if (modalInstance) {
			modalInstance.show();
		}
	}

	function goToDate() {
		$selectedDate = { year: log.year, month: log.month, day: log.day };
		if (modalInstance) {
			modalInstance.hide();
		}
	}
</script>

<!-- svelte-ignore a11y_consider_explicit_label -->
<button onclick={openModal} id="zoomButton" class="btn" style="height:100px;">
	<div class="d-flex flex-row h-100" style="width: 200px;">
		<div class="left d-flex flex-column justify-content-evenly px-1">
			<div><b>{log?.year}</b></div>
			<div><em><b>{log?.years_old}</b> {$t('aLookBack.Year_one_letter')}</em></div>
		</div>
		<div class="html-preview p-1">
			{@html marked.parse(log?.text)}
		</div>
	</div>
</button>

<!-- Standard Bootstrap Modal -->
<div
	bind:this={modal}
	class="modal fade"
	tabindex="-1"
	aria-labelledby="alookbackModalLabel"
	aria-hidden="true"
>
	<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable modal-lg">
		<div class="modal-content">
			<div class="modal-header">
				<h5 class="modal-title" id="alookbackModalLabel">
					{$t('aLookBack.header_X_years_ago', { years_old: log?.years_old })} | {new Date(
						log?.year,
						log?.month - 1,
						log?.day
					).toLocaleDateString($tolgee.getLanguage(), {
						weekday: 'long',
						day: '2-digit',
						month: '2-digit',
						year: 'numeric'
					})}
				</h5>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body">
				{@html marked.parse(log?.text)}
			</div>
			<div class="modal-footer">
				<button onclick={goToDate} class="btn btn-primary">
					{$t('aLookBack.open')}
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.left {
		border-top-left-radius: 0.375rem;
		border-bottom-left-radius: 0.375rem;
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
		pointer-events: none;
	}

	:global(body[data-bs-theme='dark']) .html-preview::after {
		background: linear-gradient(to bottom, transparent, rgba(80, 80, 80, 0.45) 80%);
	}

	:global(body[data-bs-theme='light']) .html-preview::after {
		background: linear-gradient(to bottom, transparent, rgba(219, 219, 219, 0.45) 80%);
	}

	:global(body[data-bs-theme='dark']) #zoomButton {
		background-color: rgba(138, 138, 138, 0.45);
		color: #ececec;
	}

	:global(body[data-bs-theme='dark']) .left {
		background-color: rgba(141, 141, 141, 0.45);
	}

	:global(body[data-bs-theme='light']) .left {
		background-color: rgba(180, 180, 180, 0.45);
	}

	#zoomButton {
		background-color: rgba(219, 219, 219, 0.45);
		transition: 0.3s ease;
		padding: 0 !important;
	}

	#zoomButton:hover {
		background-color: rgba(219, 219, 219, 0.65);
	}

	.modal-header {
		border-bottom: none;
	}

	.modal-footer {
		border-top: none;
	}
</style>
