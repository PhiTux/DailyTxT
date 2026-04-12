<script>
	import * as bootstrap from 'bootstrap';
	import axios from 'axios';
	import { API_URL } from '$lib/APIurl.js';
	import { getTranslate } from '@tolgee/svelte';
	import { onDestroy, onMount } from 'svelte';
	import { marked } from 'marked';
	import Tag from '$lib/Tag.svelte';
	import { tags } from '$lib/tagStore.js';
	import { selectedDate, cal } from '$lib/calendarStore';
	import { readingMode } from '$lib/settingsStore.js';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	const { t } = getTranslate();

	const renderer = {
		link(href, title, text) {
			const link = marked.Renderer.prototype.link.call(this, href, title, text);
			return link.replace('<a', "<a target='_blank' rel='noreferrer' ");
		}
	};
	marked.use({
		renderer: renderer,
		breaks: true,
		gfm: true
	});

	let {
		open = $bindable(false),
		day = null,
		month = null,
		year = null,
		language = 'en'
	} = $props();

	let loading = $state(false);
	let modalElement;
	let modalInstance = null;
	let dayLog = $state({
		text: '',
		date_written: '',
		tags: [],
		files: []
	});

	onMount(() => {
		if (!modalElement) return;

		modalInstance = new bootstrap.Modal(modalElement, {
			backdrop: true,
			keyboard: true,
			focus: true
		});

		const handleHidden = () => {
			open = false;
		};

		modalElement.addEventListener('hidden.bs.modal', handleHidden);

		onDestroy(() => {
			modalElement.removeEventListener('hidden.bs.modal', handleHidden);
			modalInstance?.dispose();
			modalInstance = null;
		});
	});

	function openDayLog() {
		const nextDate = {
			day: parseInt(day, 10),
			month: parseInt(month, 10),
			year: parseInt(year, 10)
		};

		$selectedDate = nextDate;
		$cal.currentMonth = nextDate.month - 1;
		$cal.currentYear = nextDate.year;
		$readingMode = false;
		goto(resolve('/write'));
	}

	$effect(() => {
		if (!modalInstance) return;

		if (open) {
			modalInstance.show();
		} else {
			modalInstance.hide();
		}
	});

	$effect(() => {
		if (!open) return;
		if (!day || !month || !year) {
			return;
		}
		loadDayLog();
	});

	function closeModal() {
		modalInstance?.hide();
	}

	function loadDayLog() {
		loading = true;

		axios
			.get(`${API_URL}/logs/getLog`, {
				params: { year, month, day }
			})
			.then((response) => {
				dayLog = {
					text: response.data?.text || '',
					date_written: response.data?.date_written || '',
					tags: Array.isArray(response.data?.tags) ? response.data.tags : [],
					files: Array.isArray(response.data?.files) ? response.data.files : [],
					pins: Array.isArray(response.data?.pins) ? response.data.pins : []
				};
			})
			.catch((error) => {
				console.error('Error loading day preview:', error);

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingDayPreview'));
				toast.show();
			})
			.finally(() => {
				loading = false;
			});
	}
</script>

<div class="modal fade" tabindex="-1" aria-hidden="true" bind:this={modalElement}>
	<div class="modal-dialog modal-dialog-centered preview-modal-dialog">
		<div class="modal-content preview-modal shadow-lg">
			<div class="modal-header">
				<h5 class="modal-title">
					{new Date(year, month - 1, day).toLocaleDateString(language, {
						year: 'numeric',
						month: '2-digit',
						day: '2-digit'
					})}
				</h5>
				<button type="button" class="btn-close" aria-label="Close" onclick={closeModal}></button>
			</div>

			<div class="modal-body">
				{#if loading}
					<div class="d-flex align-items-center gap-2">
						<div class="spinner-border spinner-border-sm" role="status"></div>
					</div>
				{:else}
					<div class="preview-text">
						{#if dayLog.text}
							<!-- eslint-disable-next-line svelte/no-at-html-tags-->
							{@html marked.parse(dayLog.text)}
						{:else}
							<em>{$t('log.no_entry')}</em>
						{/if}
					</div>

					{#if dayLog.tags.length > 0}
						<div class="mt-3">
							<div class="d-flex flex-wrap gap-1">
								{#each dayLog.tags as tag, i (tag + '-' + i)}
									<Tag tag={$tags.filter((t) => t.id === tag)[0]} />
								{/each}
							</div>
						</div>
					{/if}
					<div class="border-top mt-2 pt-2 gap-2 d-flex flex-wrap">
						{#if dayLog.files.length > 0}
							<div class="badge rounded-pill text-bg-light">
								{$t('settings.statistics.fileCount', { fileCount: dayLog.files.length })}
							</div>
						{/if}
						{#if dayLog.pins?.length > 0}
							<div class="badge rounded-pill text-bg-light">
								{$t('settings.statistics.pinCount', { pinCount: dayLog.pins?.length })}
							</div>
						{/if}
					</div>
				{/if}
				<div class="d-flex flex-row justify-content-center">
					<button class="btn btn-primary" onclick={openDayLog}>{$t('aLookBack.open')}</button>
				</div>
			</div>
		</div>
	</div>
</div>

<div class="toast-container position-fixed bottom-0 end-0 p-3">
	<div
		id="toastErrorLoadingDayPreview"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_loading_day_preview')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>
</div>

<style>
	.preview-modal-dialog {
		max-width: min(700px, 96vw);
	}

	.preview-modal {
		max-height: min(80vh, 760px);
		overflow: auto;
	}

	.preview-text {
		white-space: normal;
		word-break: break-word;
		line-height: 1.4;
	}

	.modal-content {
		border: none !important;
	}
</style>
