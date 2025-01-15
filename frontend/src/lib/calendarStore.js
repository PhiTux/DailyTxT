import {writable} from 'svelte/store';

let date = new Date();

export let selectedDate = writable(date);

export let cal = writable({
  daysWithLogs: [],
  daysWithFiles: [],
  currentMonth: date.getMonth(),
  currentYear: date.getFullYear(),
});

export let readingDate = writable(date)