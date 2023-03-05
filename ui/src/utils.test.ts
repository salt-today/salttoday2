import type { Filters } from '@src/types';
import parseQueryParams from './utils';

it('parssQueryParams correctly', async () => {
	const filter: Filters = {};
	const obj = {
		page: 1,
		itemsPerPage: 10,
		...filter
	};

	expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10');
});
