<script>
	import Ssr from '@src/examples/ssr.svelte';
	import Header from '../lib/header.svelte';
	import WinnerCard from '../lib/winner-card.svelte';
</script>

<Header />
<div class="container">
	<div class="flex mt-10">
		<div class="basis-1/3">
			<WinnerCard title="3rd Place" initials="SP" name="Sad Panda" />
		</div>
		<div class="basis-1/3">
			<WinnerCard title="1st Place" initials="CK" name="Captain Kangaroo" />
		</div>
		<div class="basis-1/3">
			<WinnerCard title="2nd Place" initials="HK" name="Hello Kitty" />
		</div>
	</div>
	<div>
		<Ssr />
	</div>
</div>
