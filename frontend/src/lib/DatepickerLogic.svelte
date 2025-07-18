<script>
	import { API_URL } from '$lib/APIurl.js';
	import { cal } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { currentUser } from '$lib/helpers.js';

	$effect(() => {
		if ($cal.currentMonth || $cal.currentYear) {
			loadMarkedDays();
		}
	});

	let lastMonth = $cal.currentMonth - 1;
	let lastYear = $cal.currentYear;
	let isLoadingMarkedDays = false;
	function loadMarkedDays() {
		if ($cal.currentMonth === lastMonth && $cal.currentYear === lastYear) {
			return;
		}

		if (!$currentUser) {
			console.log('User not logged in, skipping loadMarkedDays');
			return;
		}

		if (isLoadingMarkedDays) {
			return;
		}
		isLoadingMarkedDays = true;

		axios
			.get(API_URL + '/logs/getMarkedDays', {
				params: {
					month: $cal.currentMonth + 1,
					year: $cal.currentYear
				}
			})
			.then((response) => {
				$cal.daysWithLogs = [...response.data.days_with_logs];
				$cal.daysWithFiles = [...response.data.days_with_files];
				$cal.daysBookmarked = [...response.data.days_bookmarked];
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				lastMonth = $cal.currentMonth;
				lastYear = $cal.currentYear;
				isLoadingMarkedDays = false;
			});
	}
</script>
