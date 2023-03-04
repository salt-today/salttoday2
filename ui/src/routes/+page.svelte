<script>
	import { Navbar, NavBrand, NavLi, NavUl, NavHamburger, Avatar, Dropdown, DropdownItem, DropdownHeader, DropdownDivider } from 'flowbite-svelte'
</script>

<Navbar let:hidden let:toggle color="blue">
    <NavBrand href="/">
      <img src="salt-shaker.png" class="mr-3 h-6 sm:h-9" alt="Flowbite Logo"/>
      <span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white">Salt Mines</span>
    </NavBrand>
    <div class="flex items-center md:order-2">
      <Avatar id="avatar-menu">SM</Avatar>
      <NavHamburger on:click={toggle} class1="w-full md:flex md:w-auto md:order-1"/>
    </div>
    <Dropdown placement="bottom" triggeredBy="#avatar-menu">
      <DropdownHeader>
      <span class="block text-sm"> Bonnie Green </span>
      <span class="block truncate text-sm font-medium"> name@flowbite.com </span>
      </DropdownHeader>
      <DropdownItem>Leaderboard</DropdownItem>
      <DropdownItem>Profile</DropdownItem>
      <DropdownDivider />
      <DropdownItem>Sign out</DropdownItem>
    </Dropdown>
  </Navbar>

<h1>Leaderboard</h1>

