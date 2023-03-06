<script>
	import {
		Avatar,
		Card,
	} from 'flowbite-svelte';

  export let initials = "";
  export let name = "";
  export let title = "";
</script>

<Card padding='sm' color="red">
    <div class="flex flex-col items-center pb-4">
      <Avatar size="lg">{initials}</Avatar>
        <h5 class="mb-1 text-xl font-medium text-gray-900 dark:text-white">{name}</h5>
        <span class="text-sm text-gray-500 dark:text-gray-400">{title}</span>
    </div>
</Card>



