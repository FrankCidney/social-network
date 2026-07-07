'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { motion } from 'framer-motion';
import {
	ArrowRight,
	Globe,
	Loader,
	Plus,
	ShieldCheck,
	Users,
} from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { type Group, groupAPI } from '@/lib/api';

type CreateGroupFormState = {
	title: string;
	description: string;
};

const INITIAL_CREATE_FORM: CreateGroupFormState = {
	title: '',
	description: '',
};

function normalizeGroupsResponse(response: Group[] | { groups?: Group[] } | null) {
	if (Array.isArray(response)) {
		return response;
	}

	if (response && Array.isArray(response.groups)) {
		return response.groups;
	}

	return [];
}

export default function GroupsPage() {
	const router = useRouter();
	const [groups, setGroups] = React.useState<Group[]>([]);
	const [isLoading, setIsLoading] = React.useState(true);
	const [error, setError] = React.useState<string | null>(null);
	const [isCreateOpen, setIsCreateOpen] = React.useState(false);
	const [createForm, setCreateForm] = React.useState(INITIAL_CREATE_FORM);
	const [createError, setCreateError] = React.useState<string | null>(null);
	const [isCreating, setIsCreating] = React.useState(false);

	React.useEffect(() => {
		const fetchGroups = async () => {
			try {
				const response = await groupAPI.getGroups();
				setGroups(normalizeGroupsResponse(response));
			} catch (err) {
				setError(err instanceof Error ? err.message : 'Failed to load groups');
			} finally {
				setIsLoading(false);
			}
		};

		void fetchGroups();
	}, []);

	const handleCreateGroup = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();

		const title = createForm.title.trim();
		const description = createForm.description.trim();

		if (!title) {
			setCreateError('A group title is required.');
			return;
		}

		setIsCreating(true);
		setCreateError(null);

		try {
			const createdGroup = await groupAPI.createGroup({ title, description });
			setGroups((current) => [createdGroup, ...current]);
			setCreateForm(INITIAL_CREATE_FORM);
			setIsCreateOpen(false);
			router.push(`/groups/${createdGroup.id}`);
		} catch (err) {
			setCreateError(err instanceof Error ? err.message : 'Failed to create group');
		} finally {
			setIsCreating(false);
		}
	};

	if (isLoading) {
		return (
			<div className="flex min-h-[24rem] items-center justify-center rounded-[28px] border border-gray-100 bg-white">
				<Loader className="h-8 w-8 animate-spin text-indigo-600" />
			</div>
		);
	}

	if (error) {
		return (
			<div className="rounded-[28px] border border-red-200 bg-red-50 p-6 text-red-600">
				<p className="font-bold">Error loading groups</p>
				<p className="mt-1 text-sm">{error}</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<section className="overflow-hidden rounded-[28px] border border-gray-100 bg-white">
				<div className="px-6 py-7 lg:px-8">
					<div className="space-y-5">
						<div className="flex flex-wrap items-center justify-between gap-4">
							<div>
								<h1 className="text-3xl font-bold tracking-tight text-gray-900 sm:text-4xl">
									Groups
								</h1>
							</div>
							<Button
								type="button"
								variant={isCreateOpen ? 'secondary' : 'primary'}
								size="md"
								onClick={() => {
									setIsCreateOpen((current) => !current);
									setCreateError(null);
								}}
								className="shrink-0"
							>
								<Plus className="mr-2 h-4 w-4" />
								{isCreateOpen ? 'Close' : 'Create'}
							</Button>
						</div>

						{isCreateOpen ? (
							<form
								className="mt-5 space-y-4 rounded-[24px] border border-indigo-100 bg-gradient-to-br from-indigo-50 via-white to-sky-50 p-5"
								onSubmit={handleCreateGroup}
							>
								<Input
									label="Group title"
									value={createForm.title}
									onChange={(event) =>
										setCreateForm((current) => ({ ...current, title: event.target.value }))
									}
									placeholder="Weekend hikers"
									maxLength={80}
								/>
								<div className="space-y-1.5">
									<label className="ml-1 text-sm font-medium text-gray-700">Description</label>
									<textarea
										value={createForm.description}
										onChange={(event) =>
											setCreateForm((current) => ({
												...current,
												description: event.target.value,
											}))
										}
										placeholder="What brings this community together?"
										rows={4}
										className="w-full rounded-bento border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-primary"
									/>
								</div>
								{createError ? <p className="text-sm text-rose-600">{createError}</p> : null}
								<div className="flex flex-wrap items-center gap-3">
									<Button type="submit" disabled={isCreating}>
										{isCreating ? <Loader className="mr-2 h-4 w-4 animate-spin" /> : <Plus className="mr-2 h-4 w-4" />}
										Create group
									</Button>
									<p className="text-xs text-gray-500">
										The creator is automatically added as a member by the backend.
									</p>
								</div>
							</form>
						) : null}
					</div>
				</div>
			</section>

			<section className="grid gap-4 xl:grid-cols-2">
				{groups.length > 0 ? (
					groups.map((group, index) => (
						<motion.article
							key={group.id}
							initial={{ opacity: 0, y: 18 }}
							animate={{ opacity: 1, y: 0 }}
							transition={{ delay: index * 0.04 }}
							className="rounded-[28px] border border-gray-100 bg-white p-6 shadow-sm transition-colors hover:border-indigo-100"
						>
							<div className="flex items-start justify-between gap-4">
								<div className="flex min-w-0 items-start gap-4">
									<div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-[20px] bg-gradient-to-br from-indigo-500 to-sky-500 text-lg font-bold text-white">
										{group.title.charAt(0).toUpperCase()}
									</div>
									<div className="min-w-0 space-y-3">
										<div className="flex flex-wrap items-center gap-2">
											<h2 className="text-xl font-bold text-gray-900">{group.title}</h2>
											<span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600">
												<Globe className="h-3.5 w-3.5" />
												Browsable
											</span>
										</div>
										<p className="text-sm leading-6 text-gray-600">{group.description || 'No description yet.'}</p>
										<div className="flex flex-wrap items-center gap-2 text-xs text-gray-500">
											<span className="rounded-full bg-gray-100 px-2.5 py-1">
												Created {new Date(group.created_at).toLocaleDateString()}
											</span>
											<span className="rounded-full bg-indigo-50 px-2.5 py-1 text-indigo-700">
												Creator: {group.creator_id}
											</span>
										</div>
									</div>
								</div>
								<ShieldCheck className="mt-1 h-5 w-5 shrink-0 text-indigo-400" />
							</div>

							<div className="mt-6 flex flex-wrap items-center gap-3">
								<Button
									type="button"
									variant="primary"
									onClick={() => router.push(`/groups/${group.id}`)}
								>
									Open Group
									<ArrowRight className="ml-2 h-4 w-4" />
								</Button>
							</div>
						</motion.article>
					))
				) : (
					<div className="rounded-[28px] border border-dashed border-gray-200 bg-white px-6 py-12 text-center xl:col-span-2">
						<Users className="mx-auto h-12 w-12 text-gray-300" />
						<h3 className="mt-4 text-lg font-bold text-gray-900">No groups yet</h3>
						<p className="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-500">
							Create the first group to get the community area started.
						</p>
					</div>
				)}
			</section>
		</div>
	);
}
