<script>
	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<div class="settings-admin">
	<h2 class="h4 mb-3">{$t('settings.admin.title')}</h2>
	<p class="text-muted">{$t('settings.admin.placeholder')}</p>
</div>

<style>
	.settings-admin {
		min-height: 40vh;
	}
</style>
