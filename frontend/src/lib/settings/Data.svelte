<script>
	import { slide } from 'svelte/transition';
	import { Fa } from 'svelte-fa';
	import { faDownload, faUpload } from '@fortawesome/free-solid-svg-icons';

	const curlCommand = String.raw`curl 
-X POST 
-H "Content-Type: application/json" 
-d '{
"username":"user",
"password":"password",
"encrypted":true,
"includeFiles":true,
"includeTemplates":true,
"includeTags":true,
"startDate":"",
"endDate":""
}' 
https://dailytxt.mydomain.tld/api/logs/backupUser 
-o ./backup.zip`;

	let {
		exportPeriod = $bindable(),
		exportStartDate = $bindable(),
		exportEndDate = $bindable(),
		exportSplit = $bindable(),
		exportImagesInHTML = $bindable(),
		exportTagsInHTML = $bindable(),
		exportExtendedFormatting = $bindable(),
		isExporting,
		exportData,
		backupPeriod = $bindable(),
		backupStartDate = $bindable(),
		backupEndDate = $bindable(),
		backupEncrypted = $bindable(),
		backupIncludeFiles = $bindable(),
		backupIncludeTemplates = $bindable(),
		backupIncludeTags = $bindable(),
		backupIncludeBookmarks = $bindable(),
		backupPassword = $bindable(),
		isBackingUp,
		showBackupError,
		backupData,
		importFile = $bindable(),
		importEncrypted = $bindable(),
		importPassword = $bindable(),
		isImporting,
		importFileProgress,
		importErrorMessage = $bindable(),
		showImportError,
		showImportSuccess,
		importData
	} = $props();

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<h3 class="text-primary">📁 {$t('settings.data')}</h3>
<div>
	<h5>{$t('settings.export')}</h5>
	{$t('settings.export.description')}

	<h6>{$t('settings.export.period')}</h6>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="exportPeriod"
			value="periodAll"
			id="exportPeriodAll"
			bind:group={exportPeriod}
		/>
		<label class="form-check-label" for="exportPeriodAll">{$t('settings.export.period_all')}</label>
	</div>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="exportPeriod"
			value="periodVariable"
			id="exportPeriodVariable"
			bind:group={exportPeriod}
		/>
		<label class="form-check-label" for="exportPeriodVariable">
			{$t('settings.export.period_variable')}</label
		>
		{#if exportPeriod === 'periodVariable'}
			<div class="d-flex flex-row" transition:slide>
				<div class="me-2">
					<label for="exportStartDate">{$t('settings.export.start_date')}</label>
					<input
						type="date"
						class="form-control me-2"
						id="exportStartDate"
						bind:value={exportStartDate}
					/>
				</div>
				<div>
					<label for="exportEndDate">{$t('settings.export.end_date')}</label>
					<input type="date" class="form-control" id="exportEndDate" bind:value={exportEndDate} />
				</div>
			</div>
			{#if exportStartDate !== '' && exportEndDate !== '' && exportStartDate > exportEndDate}
				<div transition:slide>
					<div class="pt-2"></div>
					<div class="alert alert-danger mb-0" role="alert">
						{$t('settings.export.period_invalid')}
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<h6>{$t('settings.export.split')}</h6>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="split"
			value="aio"
			id="splitAIO"
			bind:group={exportSplit}
		/>
		<label class="form-check-label" for="splitAIO">{$t('settings.export.split_aio')} </label>
	</div>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="split"
			value="year"
			id="splitYear"
			bind:group={exportSplit}
		/>
		<label class="form-check-label" for="splitYear">{$t('settings.export.split_year')} </label>
	</div>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="split"
			value="month"
			id="splitMonth"
			bind:group={exportSplit}
		/>
		<label class="form-check-label" for="splitMonth">{$t('settings.export.split_month')} </label>
	</div>

	<h6>{$t('settings.export.show_images')}</h6>
	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			name="images"
			id="exportImagesInHTML"
			bind:checked={exportImagesInHTML}
		/>
		<label class="form-check-label" for="exportImagesInHTML">
			{@html $t('settings.export.show_images_description')}
		</label>
	</div>

	<h6>{$t('settings.export.show_tags')}</h6>
	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="exportTagsInHTML"
			bind:checked={exportTagsInHTML}
		/>
		<label class="form-check-label" for="exportTagsInHTML">
			{$t('settings.export.show_tags_description')}
		</label>
	</div>

	<h6>{$t('settings.export.extended_formatting')}</h6>
	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="exportExtendedFormatting"
			bind:checked={exportExtendedFormatting}
		/>
		<label class="form-check-label" for="exportExtendedFormatting">
			{$t('settings.export.extended_formatting_description')}
		</label>
	</div>

	<div class="form-text">
		{@html $t('settings.export.help_text')}
	</div>
	<button
		class="btn btn-primary mt-3"
		onclick={exportData}
		data-sveltekit-noscroll
		disabled={isExporting ||
			(exportPeriod === 'periodVariable' && (exportStartDate === '' || exportEndDate === ''))}
	>
		{$t('settings.export.export_button')}
		{#if isExporting}
			<div class="spinner-border spinner-border-sm ms-2" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		{/if}
	</button>
</div>

<div>
	<h5><Fa icon={faDownload}></Fa> {$t('settings.backup')}</h5>

	{$t('settings.backup.description')}

	<h6>{$t('settings.export.period')}</h6>

	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="backupPeriod"
			value="backupPeriodAll"
			id="backupPeriodAll"
			bind:group={backupPeriod}
		/>
		<label class="form-check-label" for="backupPeriodAll">{$t('settings.export.period_all')}</label>
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="backupPeriod"
			value="backupPeriodVariable"
			id="backupPeriodVariable"
			bind:group={backupPeriod}
		/>
		<label class="form-check-label" for="backupPeriodVariable">
			{$t('settings.export.period_variable')}</label
		>
		{#if backupPeriod === 'backupPeriodVariable'}
			<div class="d-flex flex-row" transition:slide>
				<div class="me-2">
					<label for="backupStartDate">{$t('settings.export.start_date')}</label>
					<input
						type="date"
						class="form-control me-2"
						id="backupStartDate"
						bind:value={backupStartDate}
					/>
				</div>
				<div>
					<label for="backupEndDate">{$t('settings.export.end_date')}</label>
					<input type="date" class="form-control" id="backupEndDate" bind:value={backupEndDate} />
				</div>
			</div>
			{#if backupStartDate !== '' && backupEndDate !== '' && backupStartDate > backupEndDate}
				<div transition:slide>
					<div class="pt-2"></div>
					<div class="alert alert-danger mb-0" role="alert">
						{$t('settings.export.period_invalid')}
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<h6>{$t('settings.backup.encryption')}</h6>
	<div class="form-text">
		{$t('settings.backup.encryption_description')}
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="backupEncrypted"
			value={false}
			id="backupDecrypted"
			bind:group={backupEncrypted}
		/>
		<label class="form-check-label" for="backupDecrypted"
			>🔓 {$t('settings.backup.decrypted')}</label
		>
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="backupEncrypted"
			value={true}
			id="backupEncrypted"
			bind:group={backupEncrypted}
		/>
		<label class="form-check-label" for="backupEncrypted"
			>🔒 {$t('settings.backup.encrypted')}</label
		>

		{#if backupEncrypted}
			<div class="d-flex flex-row" transition:slide>
				<div class="alert alert-warning rounded-4 mt-2" role="alert">
					{$t('settings.backup.encryption_warning')}
				</div>
			</div>
		{/if}
	</div>

	<h6>{$t('settings.backup.additionalSettings')}</h6>

	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="backupIncludeFiles"
			bind:checked={backupIncludeFiles}
		/>
		<label class="form-check-label" for="backupIncludeFiles">
			{$t('settings.backup.include_files_description')}
		</label>
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="backupIncludeTemplates"
			bind:checked={backupIncludeTemplates}
		/>
		<label class="form-check-label" for="backupIncludeTemplates">
			{$t('settings.backup.include_templates_description')}
		</label>
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="backupIncludeTags"
			bind:checked={backupIncludeTags}
		/>
		<label class="form-check-label" for="backupIncludeTags">
			{$t('settings.backup.include_tags_description')}
		</label>
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="checkbox"
			id="backupIncludeBookmarks"
			bind:checked={backupIncludeBookmarks}
		/>
		<label class="form-check-label" for="backupIncludeBookmarks">
			{$t('settings.backup.include_bookmarks_description')}
		</label>
	</div>

	<form onsubmit={backupData}>
		<div class="form-floating mt-3">
			<input
				type="password"
				class="form-control"
				id="backupPassword"
				placeholder={$t('settings.password.confirm_password')}
				bind:value={backupPassword}
			/>
			<label for="backupPassword">{$t('settings.password.confirm_password')}</label>
		</div>
	</form>

	<button
		class="btn btn-primary mt-3"
		onclick={backupData}
		data-sveltekit-noscroll
		disabled={isBackingUp ||
			(backupPeriod === 'backupPeriodVariable' &&
				(backupStartDate === '' || backupEndDate === '')) ||
			backupPassword.trim() === ''}
	>
		{$t('settings.backup.backup_button')}
		{#if isBackingUp}
			<div class="spinner-border spinner-border-sm ms-2" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		{/if}
	</button>

	{#if showBackupError}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.backup.backup_error')}
		</div>
	{/if}

	<hr />

	<div class="accordion mt-2" id="APIaccordion">
		<div class="accordion-item">
			<h2 class="accordion-header">
				<button
					class="accordion-button collapsed"
					type="button"
					data-bs-toggle="collapse"
					data-bs-target="#collapseOne"
					aria-expanded="false"
					aria-controls="collapseOne"
				>
					{$t('settings.backup.API_endpoint')}
				</button>
			</h2>
			<div id="collapseOne" class="accordion-collapse collapse" data-bs-parent="#APIaccordion">
				<div class="accordion-body">
					<p style="text-decoration: underline">Example:</p>

					<pre><code class="text-danger-emphasis">{@html curlCommand}</code></pre>

					// Note: Empty startDate and endDate will backup all data.<br />
					// Otherwise, use format "YYYY-MM-DD".
				</div>
			</div>
		</div>
	</div>
</div>

<div>
	<h5><Fa icon={faUpload}></Fa> {$t('settings.import')}</h5>

	{@html $t('settings.import.description')}

	<div class="mt-3 mb-3">
		<label for="importFile" class="form-label"><h6>{$t('settings.import.select_file')}</h6></label>
		<input bind:files={importFile} class="form-control" type="file" accept=".zip" id="importFile" />
	</div>

	<h6>{$t('settings.backup.encryption')}</h6>
	<div class="form-text">
		{$t('settings.import.encryption_description')}
	</div>

	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="importEncrypted"
			value={false}
			id="importDecrypted"
			bind:group={importEncrypted}
		/>
		<label class="form-check-label" for="importDecrypted"
			>🔓 {$t('settings.backup.decrypted')}</label
		>
	</div>

	{#if importEncrypted === false}
		<div class="" transition:slide>
			<div class="pt-2 pb-2">
				<div class="alert alert-warning" style="margin-bottom: 0 !important" role="alert">
					{$t('settings.import.import_decrypted_warning')}<br />
					<code>{importErrorMessage}</code>
				</div>
			</div>
		</div>
	{/if}

	<div class="form-check pt-1">
		<input
			class="form-check-input"
			type="radio"
			name="importEncrypted"
			value={true}
			id="importEncrypted"
			bind:group={importEncrypted}
		/>
		<label class="form-check-label" for="importEncrypted"
			>🔒 {$t('settings.backup.encrypted')}</label
		>
	</div>

	{#if importEncrypted === true}
		<div transition:slide>
			<div class="pt-2">
				<form onsubmit={importData}>
					<div class="form-floating">
						<input
							type="password"
							class="form-control"
							id="importPassword"
							placeholder={$t('settings.import.password')}
							bind:value={importPassword}
						/>
						<label for="importPassword">{$t('settings.import.password')}</label>
					</div>
				</form>
			</div>
		</div>
	{/if}

	{#if isImporting}
		<div transition:slide>
			<div class="pt-3">
				<div
					class="progress"
					role="progressbar"
					aria-label="Upload progress"
					aria-valuemin="0"
					aria-valuemax="100"
				>
					<div
						class="progress-bar {importFileProgress === 100
							? 'progress-bar-striped progress-bar-animated'
							: ''}"
						style:width={importFileProgress + '%'}
					>
						{#if importFileProgress !== 100}
							{$t('settings.import.upload_progress', {
								progress: importFileProgress
							})}
						{:else}
							{$t('settings.import.processing_file')}
						{/if}
					</div>
				</div>
			</div>
		</div>
	{/if}

	<button
		class="btn btn-primary mt-3"
		onclick={importData}
		data-sveltekit-noscroll
		disabled={isImporting ||
			!importFile ||
			(importEncrypted && importPassword.trim() === '') ||
			importEncrypted === ''}
	>
		{$t('settings.import.import_button')}
		{#if isImporting}
			<div class="spinner-border spinner-border-sm ms-2" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
		{/if}
	</button>

	{#if showImportError}
		<div class="pt-2" transition:slide>
			<div class="alert alert-danger" role="alert">
				{$t('settings.import.import_error')}:<br />
				<code>{importErrorMessage}</code>
			</div>
		</div>
	{/if}
	{#if showImportSuccess}
		<div class="pt-2" transition:slide>
			<div class="alert alert-success" role="alert">
				{@html $t('settings.import.import_success')}
			</div>
		</div>
	{/if}
</div>
