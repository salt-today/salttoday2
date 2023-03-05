import Ssr from './ssr.svelte';
import { render, fireEvent } from '@testing-library/svelte';
import axios from 'axios';
jest.mock('axios');

it('it works', async () => {
	const result = render(Ssr);
	const res = result.getByTestId('filters-menu-tid');
	expect(res).toBeTruthy();
	expect(axios.get).toHaveBeenCalled();
});
