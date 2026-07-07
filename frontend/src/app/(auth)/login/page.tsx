'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { authAPI, LoginPayload } from '@/lib/api';

export default function LoginPage() {
	const router = useRouter();
	const searchParams = useSearchParams();
	const nextPath = searchParams.get('next') || '/feed';

	const [formData, setFormData] = React.useState<LoginPayload>({
		email: '',
		password: '',
	});

	const [isLoading, setIsLoading] = React.useState(false);
	const [error, setError] = React.useState('');

	const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		setFormData((prev) => ({
			...prev,
			[e.target.name]: e.target.value,
		}));
	};

	const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();

		setError('');
		setIsLoading(true);

		try {
			await authAPI.login(formData);

			router.push(nextPath);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: 'Login failed. Please try again.'
			);
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<div className="space-y-6">
			<div className="space-y-2">
				<h2 className="text-2xl font-bold tracking-tight text-text-main">Welcome back</h2>
				<p className="text-sm text-gray-500">
					Enter your credentials to access your account
				</p>
			</div>

			<form onSubmit={handleSubmit} className="space-y-4">
				<Input
					label="Email"
					placeholder="name@example.com"
					type="email"
					name="email"
					value={formData.email}
					onChange={handleChange}
					required
					autoComplete="email"
				/>
				<div className="space-y-1">
					<Input
						label="Password"
						placeholder="••••••••"
						type="password"
						name="password"
						value={formData.password}
						onChange={handleChange}
						required
						autoComplete="current-password"
					/>
					<div className="flex justify-end">
					</div>
				</div>

				{error && (
					<div className="rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-600">
						{error}
					</div>
				)}

				<Button
					type="submit"
					className="w-full"
					disabled={isLoading}
				>
					{isLoading ? 'Signing in...' : 'Sign In'}
				</Button>
			</form>

			<div className="text-center text-sm">
				<span className="text-gray-500">Don&apos;t have an account? </span>
				<Link
					href={nextPath === '/feed' ? '/register' : `/register?next=${encodeURIComponent(nextPath)}`}
					className="font-medium text-primary hover:underline"
				>
					Create an account
				</Link>
			</div>
		</div>
	);
}
