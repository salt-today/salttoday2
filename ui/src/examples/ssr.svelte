<script lang="ts">
	import { onMount } from 'svelte';
	import axios from 'axios';
	import parseQueryParams from '@src/utils/utils';
	import type { Filters, Comment } from '@src/types/types';

	let includeUsername = false;
	let authorNames = '';

	const ITEMS_PER_PAGE = 10;
	let currentPage = 1;
	let totalComments = 0;
	let comments: Comment[] = [];
	let currentFilters: Filters = {};

	// Fetch the comments from the scraper by the page & ITEMS_PER_PAGE. Also pass along any filtering provided by the user in the request.
	async function fetchComments({ page = 1, filters = {} } = {}) {
		const queryParams = parseQueryParams({
			page,
			itemsPerPage: ITEMS_PER_PAGE,
			...filters
		});
		// const queryParams = '';
		// FIXME: use corrected endpoint
		const response = await axios.get(`/api/v1/comments?${queryParams}`);
		currentPage = page;
		// totalComments = response.data.totalComments;
		totalComments = 20;
		// comments = response.data.comments.map((c: any) => ({
		comments = Array.from(
			{ length: totalComments },
			(v, i): Comment => ({
				userId: i,
				text: `${i}`,
				time: new Date(Date.now()),
				name: `${i}`,
				articleId: i,
				likes: Math.random() * (i ^ 2) * 1000,
				dislikes: Math.random() * (i ^ 2) * 20,
				id: i
			})
		);
	}
	onMount(() => fetchComments({ page: 1 }));
</script>

<div>
	<ul>
		{#each comments as comment}
			<li>{comment.userId}</li>
		{/each}
	</ul>

	<ul data-testid={'filters-menu-tid'}>
		<button
			on:click={() => {
				currentFilters.liked = !currentFilters.liked ?? true;
				console.log({ currentFilters });
			}}>Liked</button
		>
		<button
			on:click={() => {
				currentFilters.disliked = !currentFilters.disliked ?? true;
				console.log({ currentFilters });
			}}>Disliked</button
		>
		<!-- <form id={'name-input-form'} on:submit={() => {
            if (includeUserName) {
                currentFilters.author=authorNames;
            }
        }}>
            <checkbox checked={includeUserName} on:change={() => includeUserName = !includeUserName}>
                <label for={'name-input'}>Username:</label>
            <input required={includeUserName} on:change={(v: HTMLInputElement) => authorNames = v.target.value} type={'text'} id={'name-input'} name={'name-input-form'} maxlength={32} size={48}/>
            </checkbox>
        </form> -->

		<!-- FIXME: ADD Date picker-->
	</ul>

	<p>{JSON.stringify(currentFilters, undefined, 2)}</p>

	<nav>
		{#if currentPage > 1}
			<button on:click={() => fetchComments({ page: currentPage - 1, filters: currentFilters })}
				>Previous</button
			>
		{/if}

		{#if currentPage < Math.ceil(totalComments / ITEMS_PER_PAGE)}
			<button on:click={() => fetchComments({ page: currentPage - 1, filters: currentFilters })}
				>Next</button
			>
		{/if}
	</nav>
</div>
