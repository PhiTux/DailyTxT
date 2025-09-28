<script>
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal, selectedDate, readingDate } from '$lib/calendarStore.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import Sidenav from '$lib/Sidenav.svelte';
	import { onMount } from 'svelte';
	import { marked } from 'marked';
	import Tag from '$lib/Tag.svelte';
	import { tags, tagsLoaded } from '$lib/tagStore.js';
	import FileList from '$lib/FileList.svelte';
	import { autoLoadImagesThisDevice, settings } from '$lib/settingsStore';
	import { faCloudArrowDown } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import ImageViewer from '$lib/ImageViewer.svelte';
	import { alwaysShowSidenav } from '$lib/helpers.js';
	import { getTranslate, getTolgee } from '@tolgee/svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	marked.use({
		breaks: true,
		gfm: true
	});

	let logs = $state([]);

	let scrollAreaEl;

	function updateReadingDateFromScroll() {
		if (!scrollAreaEl) return;
		const logsEls = scrollAreaEl.querySelectorAll('.log');
		const containerTop = scrollAreaEl.getBoundingClientRect().top;
		let candidate = null;
		for (const el of logsEls) {
			const rect = el.getBoundingClientRect();
			// First element whose bottom edge is below the top edge of the container
			if (rect.bottom > containerTop + 4) {
				// +4px tolerance to reduce flicker
				candidate = el;
				break;
			}
		}
		if (candidate) {
			const day = parseInt(candidate.getAttribute('data-log-day'));
			if (!$readingDate || $readingDate.day !== day) {
				$readingDate = {
					year: $cal.currentYear,
					month: $cal.currentMonth + 1,
					day
				};
			}
		}
	}

	let scrollRaf;
	function onScrollHandler() {
		if (scrollRaf) cancelAnimationFrame(scrollRaf);
		scrollRaf = requestAnimationFrame(updateReadingDateFromScroll);
	}

	onMount(() => {
		scrollAreaEl = document.getElementById('scrollArea');
		if (scrollAreaEl) {
			scrollAreaEl.addEventListener('scroll', onScrollHandler, { passive: true });
		}
	});

	$effect(() => {
		if ($tagsLoaded) {
			loadMonthForReading();
		}
	});

	let currentMonth = $cal.currentMonth;
	let currentYear = $cal.currentYear;
	$effect(() => {
		if ($cal.currentMonth !== currentMonth || $cal.currentYear !== currentYear) {
			cancelDownload.abort();
			cancelDownload = new AbortController();

			loadMonthForReading();
			currentMonth = $cal.currentMonth;
			currentYear = $cal.currentYear;
		}
	});

	$effect(() => {
		if ($selectedDate) {
			$cal.currentYear = $selectedDate.year;
			$cal.currentMonth = $selectedDate.month - 1;

			let el = document.querySelector(`.log[data-log-day="${$selectedDate.day}"]`);
			if (el) {
				el.scrollIntoView({ behavior: 'smooth', block: 'start' });
			}
		}
	});

	const imageExtensions = ['jpeg', 'jpg', 'gif', 'png', 'webp'];
	//TODO: support svg? -> minsize is necessary...

	// copy of files, which are images
	$effect(() => {
		if (logs) {
			logs.forEach((log) => {
				if (log.files) {
					if (!log.images) {
						log.images = [];
					}

					log.files.forEach((file) => {
						if (
							imageExtensions.includes(file.filename.split('.').pop().toLowerCase()) &&
							!log.images.find((image) => image.uuid_filename === file.uuid_filename)
						) {
							log.images = [...log.images, file];

							if (autoLoadImages) {
								loadImage(file.uuid_filename);
							}
						}
					});
				}
			});
		}
	});

	let autoLoadImages = $derived(
		($settings.setAutoloadImagesPerDevice && $autoLoadImagesThisDevice) ||
			(!$settings.setAutoloadImagesPerDevice && $settings.autoloadImagesByDefault)
	);

	function loadImage(uuid) {
		for (let i = 0; i < logs.length; i++) {
			let log = logs[i];

			// skip log if file not in this day/log
			if (!log.images) {
				continue;
			}
			let image = log.images.find((image) => image.uuid_filename === uuid);
			if (!image) {
				continue;
			}

			log.images = log.images.map((image) => {
				if (image.uuid_filename === uuid) {
					image.loading = true;
				}
				return image;
			});

			axios
				.get(API_URL + '/logs/downloadFile', {
					params: { uuid: uuid },
					responseType: 'blob',
					signal: cancelDownload.signal
				})
				.then((response) => {
					const url = URL.createObjectURL(new Blob([response.data]));
					log.images = log.images.map((image) => {
						if (image.uuid_filename === uuid) {
							image.src = url;
							image.loading = false;
						}
						return image;
					});

					log.files = log.files.map((file) => {
						if (file.uuid_filename === uuid) {
							file.src = url;
						}
						return file;
					});
				})
				.catch((error) => {
					if (error.name == 'CanceledError') {
						return;
					}

					console.error(error);
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
					toast.show();
				});
		}
	}

	function loadImages() {
		for (let i = 0; i < logs.length; i++) {
			let log = logs[i];

			// skip log if no images in this day/log
			if (!log.images) {
				continue;
			}

			log.images.forEach((image) => {
				if (!image.src) {
					loadImage(image.uuid_filename);
				}
			});
		}
	}

	let cancelDownload = new AbortController();

	function downloadFile(uuid) {
		for (let i = 0; i < logs.length; i++) {
			let log = logs[i];

			// skip log if file not in this day/log
			if (!log.files) {
				continue;
			}
			let file = log.files.find((file) => file.uuid_filename === uuid);
			if (!file) {
				continue;
			}

			// check if src is present in files
			if (file.src) {
				triggerAutomaticDownload(uuid);
				return;
			}

			// otherwise: download from server
			log.files = log.files.map((f) => {
				if (f.uuid_filename === uuid) {
					f.downloadProgress = 0;
				}
				return f;
			});

			const config = {
				params: { uuid: uuid },
				onDownloadProgress: (progressEvent) => {
					log.files = log.files.map((file) => {
						if (file.uuid_filename === uuid) {
							file.downloadProgress = Math.round((progressEvent.loaded / file.size) * 100);
						}
						return file;
					});
				},
				signal: cancelDownload.signal,
				responseType: 'blob'
			};

			axios
				.get(API_URL + '/logs/downloadFile', {
					...config
				})
				.then((response) => {
					const url = URL.createObjectURL(new Blob([response.data]));
					log.files = log.files.map((f) => {
						if (f.uuid_filename === uuid) {
							f.src = url;
						}
						return f;
					});
				})
				.catch((error) => {
					if (error.name == 'CanceledError') {
						return;
					}

					console.error(error);
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
					toast.show();
				})
				.finally(() => {
					// remove progress
					log.files = log.files.map((f) => {
						if (f.uuid_filename === uuid) {
							f.downloadProgress = -1;
						}
						return f;
					});

					triggerAutomaticDownload(uuid);
				});
		}
	}

	// TODO adjust
	function triggerAutomaticDownload(uuid) {
		for (let i = 0; i < logs.length; i++) {
			let log = logs[i];

			// skip log if file not in this day/log
			if (!log.files) {
				continue;
			}
			let file = log.files.find((file) => file.uuid_filename === uuid);
			if (!file) {
				continue;
			}

			const a = document.createElement('a');
			a.href = file.src;
			a.download = file.filename;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
		}
	}

	let isLoadingMonthForReading = false;

	function loadMonthForReading() {
		if (isLoadingMonthForReading) {
			return;
		}
		isLoadingMonthForReading = true;

		axios
			.get(API_URL + '/logs/loadMonthForReading', {
				params: {
					month: $cal.currentMonth + 1,
					year: $cal.currentYear
				}
			})
			.then((response) => {
				logs = response.data.sort((a, b) => a.day - b.day);
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isLoadingMonthForReading = false;

				// Wait until DOM is stable (layout after markup injection) then do first calculation
				requestAnimationFrame(() => {
					updateReadingDateFromScroll();
					// Ensure an initial highlight or jump to selectedDate after month switch
					if (logs && logs.length > 0) {
						const firstDay = logs[0].day;
						const selectedMatchesMonth =
							$selectedDate &&
							$selectedDate.year === $cal.currentYear &&
							$selectedDate.month === $cal.currentMonth + 1;

						if (selectedMatchesMonth) {
							const selEl = document.querySelector(`.log[data-log-day="${$selectedDate.day}"]`);
							if (selEl) {
								// Instant jump (avoid double smooth scroll chains)
								selEl.scrollIntoView({ behavior: 'instant', block: 'start' });
								$readingDate = {
									year: $cal.currentYear,
									month: $cal.currentMonth + 1,
									day: $selectedDate.day
								};
							}
						}

						// Fallback: if readingDate not yet in this month (and no matching selectedDate), mark first log day
						if (
							(!$readingDate ||
								$readingDate.month !== $cal.currentMonth + 1 ||
								$readingDate.year !== $cal.currentYear) &&
							!selectedMatchesMonth
						) {
							$readingDate = {
								year: $cal.currentYear,
								month: $cal.currentMonth + 1,
								day: firstDay
							};
						}
					}
				});
			});
	}
</script>

<DatepickerLogic />

<!-- shown on small Screen, when triggered -->
<div class="offcanvas offcanvas-start p-3" id="sidenav" tabindex="-1">
	<div class="offcanvas-header">
		<button
			type="button"
			class="btn-close"
			data-bs-dismiss="offcanvas"
			data-bs-target="#sidenav"
			aria-label="Close"
		></button>
	</div>
	<Sidenav />
</div>

<div class="layout-read d-flex flex-row justify-content-between container-xxl">
	<!-- shown on large Screen -->
	{#if $alwaysShowSidenav}
		<div class="sidenav p-3">
			<Sidenav />
		</div>
	{/if}

	<!-- Center -->
	<div class="d-flex flex-column my-4 ms-4 flex-fill overflow-y-auto" id="scrollArea">
		{#if logs.length > 0}
			{#each logs as log (log.day)}
				<!-- Log-Area -->
				{#if ('text' in log && log.text !== '') || log.tags?.length > 0 || log.files?.length > 0}
					<div class="log glass mb-3 p-3 d-flex flex-row" data-log-day={log.day}>
						<div class="date me-3 d-flex flex-column align-items-center">
							<p class="dateNumber">{log.day}</p>
							<p class="dateDay">
								<b>
									{new Date($cal.currentYear, $cal.currentMonth, log.day).toLocaleDateString(
										$tolgee.getLanguage(),
										{
											weekday: 'long'
										}
									)}
								</b>
							</p>
							<p class="dateMonthYear">
								<i
									>{new Date($cal.currentYear, $cal.currentMonth, log.day).toLocaleDateString(
										$tolgee.getLanguage(),
										{ year: 'numeric', month: 'long' }
									)}</i
								>
							</p>
						</div>
						<div class="logContent flex-grow-1 d-flex flex-row">
							<div class="flex-grow-1 middle">
								{#if log.text && log.text !== ''}
									<div class="text">
										{@html marked.parse(log.text)}
									</div>
								{/if}
								{#if log.tags?.length > 0}
									<div class="tags d-flex flex-row flex-wrap">
										{#each log.tags as t}
											<Tag tag={$tags.find((tag) => tag.id === t)} />
										{/each}
									</div>
								{/if}
								{#if log.images?.length > 0}
									{#if !autoLoadImages && log.images.find((image) => !image.src && !image.loading)}
										<div class="d-flex flex-row">
											<button type="button" class="loadImageBtn" onclick={() => loadImages()}>
												<Fa icon={faCloudArrowDown} class="me-2" size="2x" fw /><br />
												{$t('read.load_images')}
											</button>
										</div>
									{:else}
										<ImageViewer images={log.images} />
									{/if}
								{/if}
							</div>

							{#if log.files && log.files.length > 0}
								<div class="d-flex flex-column ms-3 files">
									<FileList files={log.files} {downloadFile} />
								</div>
							{/if}
						</div>
					</div>
				{/if}
			{/each}
		{:else}
			<div class="d-flex align-items-center justify-content-center h-100">
				<div class="glass p-5 rounded-5 no-entries">
					<span id="no-entries">{$t('read.no_entries')}</span>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	#no-entries {
		font-size: 1.5rem;
		font-weight: 600;
		opacity: 0.7;
	}

	.layout-read {
		height: 100%;
		overflow: hidden;
	}

	.sidenav {
		width: 380px;
		min-width: 380px;
		overflow-y: auto; /* independent scroll */
		max-height: 100vh; /* constrain to viewport */
		padding-right: 0.5rem;
		box-sizing: border-box;
	}

	#sidenav {
		overflow-y: auto;
	}

	.files {
		max-width: 350px;
		min-width: 250px;
	}

	.middle {
		overflow-x: auto;
	}

	.loadImageBtn {
		padding: 0.5rem 1rem;
		border: none;
		margin-top: 0.5rem;
		border-radius: 5px;
		transition: all ease 0.2s;
		background-color: #ccc;
	}

	.loadImageBtn:hover {
		background-color: #bbb;
	}

	.text {
		word-wrap: anywhere;
	}

	.tags {
		gap: 0.5rem;
	}

	.log {
		border-radius: 15px;
	}

	:global(body[data-bs-theme='dark']) .glass {
		background-color: rgba(68, 68, 68, 0.6) !important;
	}

	:global(body[data-bs-theme='light']) .glass {
		background-color: rgba(122, 122, 122, 0.6) !important;
		color: rgb(19, 19, 19);
	}

	.dateNumber {
		font-size: 3rem;
		font-weight: 600;
		font-style: italic;
		opacity: 0.5;
	}

	.dateDay {
		opacity: 0.7;
		font-size: 1.2rem;
	}

	#scrollArea {
		padding-right: 1rem;
		overflow-y: auto;
		max-height: 100vh; /* scroll area uses remaining viewport height */
	}

	@media (min-width: 1300px) and (max-width: 1450px) {
		.files {
			max-width: 250px;
		}
	}

	@media (max-width: 768px) {
		.date {
			min-width: 50px;
			flex-direction: row !important;
			align-items: end !important;
		}

		.dateDay {
			margin-left: 1rem;
		}

		.dateNumber {
			margin-top: -0.5rem;
			margin-bottom: 0;
		}

		.dateMonthYear {
			margin-left: 1rem;
			opacity: 0.7;
		}

		.log {
			flex-direction: column !important;
		}

		#scrollArea {
			margin-left: 1rem !important;
		}
	}

	@media (max-width: 1300px) {
		.logContent {
			flex-direction: column !important;
		}

		#scrollArea {
			margin-top: 1rem !important;
			margin-bottom: 1rem !important;
		}
	}

	@media (min-width: 769px) {
		.date {
			min-width: 100px;
		}

		.dateMonthYear {
			display: none;
		}
	}
</style>
