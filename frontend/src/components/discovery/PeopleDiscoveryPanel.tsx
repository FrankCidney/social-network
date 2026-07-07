'use client';

import * as React from 'react';
import Link from 'next/link';
import { Lock, MessageSquare, Search, UserRound } from 'lucide-react';
import { profileAPI, resolveAssetUrl, type UserSearchResult } from '@/lib/api';

const RESULT_LIMIT = 6;

function getDisplayName(user: UserSearchResult) {
	return user.nickname?.trim() ? user.nickname : `${user.first_name} ${user.last_name}`;
}

function getHandle(user: UserSearchResult) {
	return `@${user.first_name.toLowerCase()}${user.last_name.toLowerCase()}`;
}

export function PeopleDiscoveryPanel() {
	const [query, setQuery] = React.useState('');
	const deferredQuery = React.useDeferredValue(query);
	const [results, setResults] = React.useState<UserSearchResult[]>([]);
	const [loading, setLoading] = React.useState(true);
	const [error, setError] = React.useState<string | null>(null);

	React.useEffect(() => {
		let cancelled = false;

		const loadPeople = async () => {
			setLoading(true);
			setError(null);
			try {
				const data = await profileAPI.searchUsers(deferredQuery, { limit: RESULT_LIMIT });
				if (!cancelled) {
					setResults(Array.isArray(data) ? data : []);
				}
			} catch (err) {
				if (!cancelled) {
					setResults([]);
					setError(err instanceof Error ? err.message : 'Failed to load people');
				}
			} finally {
				if (!cancelled) {
					setLoading(false);
				}
			}
		};

		void loadPeople();

		return () => {
			cancelled = true;
		};
	}, [deferredQuery]);

	return (
		<div className="bg-white rounded-xl border border-gray-100 p-4">
			<div className="flex items-center justify-between gap-3">
				<div>
					<h3 className="font-bold text-gray-900">Find people</h3>
					<p className="mt-1 text-xs text-gray-500">Search, follow, or jump to messages.</p>
				</div>
			</div>

			<div className="relative mt-4">
				<Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
				<input
					type="text"
					value={query}
					onChange={(event) => setQuery(event.target.value)}
					placeholder="Search people..."
					className="w-full rounded-full border border-gray-200 bg-gray-50 py-2 pl-10 pr-4 text-sm text-gray-900 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
				/>
			</div>

			{error ? <p className="mt-3 text-sm text-rose-600">{error}</p> : null}

			<div className="mt-4 space-y-3">
				{loading ? (
					[1, 2, 3].map((item) => (
						<div key={item} className="flex items-center gap-3 rounded-xl border border-gray-100 bg-gray-50 p-3 animate-pulse">
							<div className="h-10 w-10 rounded-full bg-gray-200" />
							<div className="flex-1 space-y-2">
								<div className="h-3 w-24 rounded bg-gray-200" />
								<div className="h-2 w-20 rounded bg-gray-200" />
							</div>
						</div>
					))
				) : results.length > 0 ? (
					results.map((person) => {
						const avatarUrl = resolveAssetUrl(person.avatar_path);

						return (
							<div key={person.id} className="rounded-xl border border-gray-100 bg-gray-50 p-3">
								<div className="flex items-start gap-3">
									<Link href={`/profile/${person.id}`} className="flex items-start gap-3 min-w-0 flex-1">
										{avatarUrl ? (
											<img
												src={avatarUrl}
												alt={getDisplayName(person)}
												className="h-10 w-10 rounded-full object-cover"
											/>
										) : (
											<div className="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-100 text-sm font-semibold text-indigo-700">
												{`${person.first_name[0] ?? ''}${person.last_name[0] ?? ''}`.toUpperCase()}
											</div>
										)}

										<div className="min-w-0 flex-1">
											<p className="truncate text-sm font-semibold text-gray-900 hover:text-indigo-600">{getDisplayName(person)}</p>
											<p className="truncate text-xs text-gray-500">{getHandle(person)}</p>
											<div className="mt-2 flex items-center gap-2 text-[11px] text-gray-500">
												{person.is_public ? (
													<span>Public profile</span>
												) : (
													<span className="inline-flex items-center gap-1">
														<Lock className="h-3 w-3" />
														Private profile
													</span>
												)}
											</div>
										</div>
									</Link>
								</div>

								<div className="mt-3 flex gap-2">
									<Link
										href={`/profile/${person.id}`}
										className="inline-flex flex-1 items-center justify-center rounded-bento bg-primary px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
									>
										<UserRound className="mr-2 h-4 w-4" />
										View profile
									</Link>
									<Link
										href={`/messages?user=${encodeURIComponent(person.id)}`}
										className="inline-flex flex-1 items-center justify-center rounded-bento bg-transparent px-3 py-1.5 text-sm font-medium text-text-main transition-colors hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
									>
										<MessageSquare className="mr-2 h-4 w-4" />
										Message
									</Link>
								</div>
							</div>
						);
					})
				) : (
					<div className="rounded-xl border border-dashed border-gray-200 bg-white px-4 py-8 text-center">
						<p className="text-sm font-medium text-gray-900">No people found</p>
						<p className="mt-1 text-xs text-gray-500">Try a different name, nickname, or user ID.</p>
					</div>
				)}
			</div>
		</div>
	);
}
