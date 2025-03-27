function formatBytes(bytes) {
	if (!+bytes) return '0 Bytes';

	const k = 1024;
	//const dm = 2; // decimal places
	const sizes = ['B', 'KB', 'MB', 'GB'];

	const i = Math.floor(Math.log(bytes) / Math.log(k));

	return `${parseFloat((bytes / Math.pow(k, i)).toFixed(0))} ${sizes[i]}`;
}

export { formatBytes };
