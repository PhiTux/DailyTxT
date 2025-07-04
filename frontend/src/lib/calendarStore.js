import {writable} from 'svelte/store';

let date = new Date();

export let selectedDate = writable({
  day: date.getDate(),
  month: date.getMonth() + 1,
  year: date.getFullYear(),
});

export let cal = writable({
  daysWithLogs: [],
  daysWithFiles: [],
  daysBookmarked: [],
  currentMonth: date.getMonth(),
  currentYear: date.getFullYear(),
});

export let readingDate = writable({
  day: date.getDate(),
  month: date.getMonth() + 1,
  year: date.getFullYear(),
});