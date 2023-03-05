export default {
	get: jest.fn(() => Promise.resolve({ data: { totalComments: 0, comments: [] } }))
};
