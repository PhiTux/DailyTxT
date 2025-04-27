<script>
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal, selectedDate, readingDate } from '$lib/calendarStore.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import Sidenav from '$lib/Sidenav.svelte';
	import { onMount } from 'svelte';
	import { marked } from 'marked';
	import Tag from '$lib/Tag.svelte';
	import { tags } from '$lib/tagStore.js';
	import FileList from '$lib/FileList.svelte';
	import { autoLoadImagesThisDevice, settings } from '$lib/settingsStore';
	import { faCloudArrowDown } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import { fade, slide } from 'svelte/transition';
	import ImageViewer from '$lib/ImageViewer.svelte';
	import { alwaysShowSidenav } from '$lib/helpers.js';

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	marked.use({
		breaks: true,
		gfm: true
	});

	let logs = $state([]);
	let search = $state('');

	let observer;

	onMount(() => {
		loadMonthForReading();

		// Highlights automatically the day in the calendar, when the log is in the viewport
		observer = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						$readingDate = new Date(
							$cal.currentYear,
							$cal.currentMonth,
							entry.target.getAttribute('data-log-day')
						);
					}
				});
			},
			{
				root: null,
				rootMargin: '0% 0px -70% 0px',
				threshold: 0.67
			}
		);
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
			$cal.currentYear = $selectedDate.getFullYear();
			$cal.currentMonth = $selectedDate.getMonth();

			let el = document.querySelector(`.log[data-log-day="${$selectedDate.getDate()}"]`);
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

	//#TODO Anpassen
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
				logs = response.data;
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isLoadingMonthForReading = false;

				setTimeout(() => {
					document.querySelectorAll('.log').forEach((log) => {
						observer.observe(log);
					});
				}, 1000);
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

<div class="d-flex flex-row justify-content-between h-100">
	<!-- shown on large Screen -->
	{#if $alwaysShowSidenav}
		<div class="sidenav p-3">
			<Sidenav />
		</div>
	{/if}

	<!-- Center -->
	<div class="d-flex flex-column my-4 ms-4 flex-fill overflow-y-auto" id="scrollArea">
		{#each logs as log (log.day)}
			<!-- Log-Area -->
			{#if ('text' in log && log.text !== '') || log.tags?.length > 0 || log.files?.length > 0}
				<div class="log mb-3 p-3 d-flex flex-row" data-log-day={log.day}>
					<div class="date me-3 d-flex flex-column align-items-center">
						<p class="dateNumber">{log.day}</p>
						<p class="dateDay">
							<b>
								{new Date($cal.currentYear, $cal.currentMonth, log.day).toLocaleDateString(
									'locale',
									{
										weekday: 'long'
									}
								)}
							</b>
						</p>
					</div>
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
										Bilder laden
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
			{/if}
		{/each}
	</div>
</div>

<style>
	.sidenav {
		width: 380px;
		min-width: 380px;
	}

	.files {
		max-width: 350px;
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
		backdrop-filter: blur(10px) saturate(150%);
		background-color: rgba(199, 199, 201, 0.329);
		border-radius: 15px;
		border: 1px solid rgba(223, 221, 221, 0.658);
	}

	.dateNumber {
		font-size: 3rem;
		font-weight: 600;
		font-style: italic;
		opacity: 0.5;
	}

	.dateDay {
		opacity: 0.7;
	}

	#scrollArea {
		padding-right: 1rem;
	}
</style>
