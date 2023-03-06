import type { Filters } from '@src/types';
import parseQueryParams from './utils';

describe('parsesQueryParams correctly', () => {
	it('defaultParams', async () => {
		const filter: Filters = {};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10');
	});
	it('city', async () => {
		const filter: Filters = {
			city: 'ssm'
		};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&city=ssm');
	});
	it('liked', async () => {
		const filter: Filters = {
			liked: true
		};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&liked=true');
	});
	it('disliked', async () => {
		const filter: Filters = {
			disliked: true
		};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&disliked=true');
	});
	it('onlyDeleted', async () => {
		const filter: Filters = {
			onlyDeleted: true
		};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&onlyDeleted=true');
	});
	it('onlyDeleted', async () => {
		const filter: Filters = {
			onlyDeleted: true
		};
		const obj = {
			page: 1,
			itemsPerPage: 10,
			...filter
		};

		expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&onlyDeleted=true');
	});
	describe('author', () => {
		it('single', async () => {
			const filter: Filters = {
				author: 'user1'
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&author=user1');
		});
		it('multi', async () => {
			const filter: Filters = {
				author: 'user1,user2'
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&author=user1,user2');
		});
	});
	describe('since', () => {
		it('day', async () => {
			const filter: Filters = {
				since: 1
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&since=1');
		});
		it('week', async () => {
			const filter: Filters = {
				since: 7
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&since=7');
		});
		it('month', async () => {
			const filter: Filters = {
				since: 30
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&since=30');
		});
		it('year', async () => {
			const filter: Filters = {
				since: 365
			};
			const obj = {
				page: 1,
				itemsPerPage: 10,
				...filter
			};

			expect(parseQueryParams(obj)).toContain('page=1&itemsPerPage=10&since=365');
		});
	});
});
