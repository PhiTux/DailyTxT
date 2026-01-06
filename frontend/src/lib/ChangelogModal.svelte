<script>
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	let { changelog = {}, current_version = '' } = $props();
</script>

<div
	class="modal fade"
	id="changelogModal"
	tabindex="-1"
	aria-labelledby="changelogModalLabel"
	aria-hidden="true"
>
	<div
		class="modal-dialog modal-lg modal-fullscreen-lg-down modal-dialog-centered modal-dialog-scrollable"
	>
		<div class="modal-content">
			<div class="modal-header">
				<h1 class="modal-title fs-5" id="changelogModalLabel">{$t('settings.about.changelog')}</h1>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body">
				{#each Object.entries(changelog) as [version, data]}
					<div class="mb-4">
						<h5 class="d-flex justify-content-between align-items-center">
							<div>
								<span class="badge text-bg-primary">{version}</span>
								{#if version === current_version}
									<span class="badge text-bg-success">{$t('settings.about.current_version')}</span>
								{/if}
							</div>
							<small class="text-body-secondary" style="font-size: 0.8rem;">{data.date}</small>
						</h5>
						<ul class="list-group list-group-flush mt-2">
							{#each data.changes as change}
								<li class="list-group-item bg-transparent py-1 px-0 border-0">
									{change}
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal"
					>{$t('modal.close')}</button
				>
			</div>
		</div>
	</div>
</div>
