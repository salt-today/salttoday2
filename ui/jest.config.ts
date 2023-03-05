export const config = {
	transform: {
		'^.+\\.svelte$': 'svelte-jester',
		'^.+\\.{js,ts}': 'ts-jest'
	},
	moduleFileExtensions: ['js', 'ts', 'svelte'],
	moduleNameMapper: {
		'@src/(.*)$': ['<rootDir>/src/$1']
	},
	preset: 'ts-jest',
	testEnvironment: 'node'
};

export default config;
