<script lang="ts">
	import { onMount } from 'svelte';
	import axios from 'axios';
	import qs from 'qs';

	// Defines a comment returned by the scraper.
	type Comment = {
		author: string;
		content: string;
		datePosted: Date;
	};

	// Filter comments based on the following traits.
	type Filters = {
		since?: Date;
		author?: string;
		liked?: boolean;
		disliked?: boolean;
		deleted?: boolean;
	};

    let includeUserName = false;
    let authorNames = '';

	const ITEMS_PER_PAGE = 10;
	let currentPage = 1;
	let totalComments = 0;
	let comments: Comment[] = [];
	let currentFilters: Filters = {};

    // Fetch the comments from the scraper by the page & ITEMS_PER_PAGE. Also pass along any filtering provided by the user in the request.
	async function fetchComments({ page = 1, filters = currentFilters }) {
		const { since, ...rest } = filters;
		const queryParams = qs.stringify({
			page,
			ITEMS_PER_PAGE,
			since: since?.getUTCMilliseconds,
			...rest
		});
		// FIXME: use corrected endpoint
		const response = await axios.get(`/api/v1/comments?${queryParams}`);
		currentPage = page;
		totalComments = response.data.totalComments;
		comments = response.data.comments.map((c: any) => ({
			author: c.author,
			content: c.content,
			datePosted: Date.parse(c.datePosted)
		}));
	}
	onMount(() => fetchComments({ page: 1 }));
</script>

<div>
	<ul>
		{#each comments as comment}
			<li>{comment.author}</li>
		{/each}
	</ul>

    <ul>
        <button on:click={() => currentFilters.liked = true} >Liked</button>
        <button on:click={() => currentFilters.disliked = true} >Disliked</button>
        <form id={'name-input-form'} on:submit={() => {
            if (includeUserName) {
                currentFilters.author=authorNames;
            }
        }}>
            <checkbox checked={includeUserName} on:change={() => includeUserName = !includeUserName}>
                <label for={'name-input'}>Username:</label>
            <input required={includeUserName} on:change={(v: HTMLInputElement) => authorNames = v.target.value} type={'text'} id={'name-input'} name={'name-input-form'} maxlength={32} size={48}>
            </checkbox>
        </form>

        <!-- FIXME: ADD Date picker-->
    </ul>

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
