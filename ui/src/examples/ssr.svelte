<script lang="ts">
 import {onMount} from 'svelte';
 import axios from 'axios';
 import qs from 'qs';

type Comment = {
    author:string;
    content:string;
    datePosted: Date;
}

type Filters = {
    since?: Date;
    author?: string;
    liked?: boolean;
    disliked?: boolean;
    deleted?: boolean;
}

 const ITEMS_PER_PAGE = 10;
 let currentPage = 1;
 let totalComments = 0;
 let comments: Comment[] = [];
 let currentFilters:Filters = {};

 async function fetchComments({page = 1, filters = currentFilters}) {
    const {since, ...rest} = filters
    const queryParams = qs.stringify({page, ITEMS_PER_PAGE, since: since?.getUTCMilliseconds, ...rest})
    // FIXME: use corrected endpoint
    const response = await axios.get(`/api/v1/comments?${queryParams}`)
    currentPage = page;
    totalComments = response.data.totalComments;
    comments = response.data.comments.map((c: any) => ({author:c.author, content: c.content, datePosted: Date.parse(c.datePosted)}));
 }
 onMount(()=>fetchComments({page: 1}))
</script>

<div>
    <ul>
        {#each comments as comment}
            <li>{comment.author}</li>
        {/each}
    </ul>

    <nav>
        {#if currentPage > 1}
            <button on:click={() => fetchComments({page: currentPage -1, filters: currentFilters})}>Previous</button>
        {/if}

        {#if currentPage < Math.ceil(totalComments / ITEMS_PER_PAGE)}
            <button on:click={() => fetchComments({page: currentPage -1, filters: currentFilters})}>Next</button>
        {/if}
    </nav>
</div>