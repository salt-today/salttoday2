<script lang="ts">
 import {onMount} from 'svelte';
 import axios from 'axios';
 import qs from 'qs';

type Comment = {
    author:string;
    content:string;
    datePosted: Date;
}

 const ITEMS_PER_PAGE = 10;
 let currentPage = 1;
 let totalComments = 0;
 let comments: Comment[] = [];

 // FIXME: Handle since, likes, dislikes
 async function fetchComments(page = 1) {
    const queryParams = qs.stringify({page, PAGE_SIZE: ITEMS_PER_PAGE})
    // FIXME: use corrected endpoint
    const response = await axios.get(`/api/comments?${queryParams}`)
    currentPage = page;
    totalComments = response.data.totalComments;
    comments = response.data.comments.map((c: any) => ({author:c.author, content: c.content, datePosted: Date.parse(c.datePosted)}));
 }
 onMount(fetchComments)
</script>

<div>
    <ul>
        {#each comments as comment}
            <li>{comment.author}</li>
        {/each}
    </ul>

    <nav>
        {#if currentPage > 1}
            <button on:click={() => fetchComments(currentPage -1)}>Previous</button>
        {/if}

        {#if currentPage < Math.ceil(totalComments / ITEMS_PER_PAGE)}
            <button on:click={() => fetchComments(currentPage + 1)}>Next</button>
        {/if}
    </nav>
</div>