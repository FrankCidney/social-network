'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { authAPI, profileAPI } from '@/lib/api';

export default function RegisterPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = searchParams.get('next') || '/feed';

  const [formData, setFormData] = React.useState({
    first_name: '',
    last_name: '',
    email: '',
    password: '',
    dob: '',
    nickname: '',
    about_me: '',
  });
  const [avatarFile, setAvatarFile] = React.useState<File | null>(null);
  const [avatarPreviewUrl, setAvatarPreviewUrl] = React.useState<string | null>(null);

  const [isLoading, setIsLoading] = React.useState(false);
  const [error, setError] = React.useState('');
  const [success, setSuccess] = React.useState('');

  React.useEffect(() => {
    if (!avatarFile) {
      setAvatarPreviewUrl(null);
      return;
    }

    const nextPreviewUrl = URL.createObjectURL(avatarFile);
    setAvatarPreviewUrl(nextPreviewUrl);

    return () => {
      URL.revokeObjectURL(nextPreviewUrl);
    };
  }, [avatarFile]);

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    setFormData((prev) => ({
      ...prev,
      [e.target.name]: e.target.value,
    }));
  };

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setAvatarFile(e.target.files?.[0] ?? null);
  };

  const handleSubmit = async (
    e: React.FormEvent<HTMLFormElement>
  ) => {
    e.preventDefault();

    setError('');
    setSuccess('');
    setIsLoading(true);

    try {
      await authAPI.register(formData);
      if (avatarFile) {
        await profileAPI.uploadAvatar(avatarFile);
      }

      setSuccess('Registration successful! Redirecting...');

      setTimeout(() => {
        router.push(nextPath);
      }, 1500);

    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Registration failed.'
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="max-h-screen flex items-center justify-center bg-gray-30 px-0">
      <div className="w-full max-w-md bg-white rounded-xl shadow-sm border border-gray-200 p-36 sm:p-8">

        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold text-gray-900">
            Create Account
          </h1>

          <p className="text-gray-500 mt-2">
            Join the Social Network
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="space-y-5"
        >
          {/* Name row: horizontally aligned */}
          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              label="First Name"
              name="first_name"
              placeholder="firstname"
              value={formData.first_name}
              onChange={handleChange}
              required
            />

            <Input
              label="Last Name"
              name="last_name"
              placeholder="lastname"
              value={formData.last_name}
              onChange={handleChange}
              required
            />
          </div>

          {/* Everything below: vertically stacked */}
          <Input
            label="Date of Birth"
            type="date"
            name="dob"
            value={formData.dob}
            onChange={handleChange}
            required
          />

          <Input
            label="Email"
            type="email"
            name="email"
            placeholder="joel@example.com"
            value={formData.email}
            onChange={handleChange}
            required
          />

          <Input
            label="Password"
            type="password"
            name="password"
            placeholder="Enter password"
            value={formData.password}
            onChange={handleChange}
            required
          />

          <Input
            label="Nickname"
            name="nickname"
            placeholder="nickname"
            value={formData.nickname}
            onChange={handleChange}
          />

          <div className="space-y-1.5">
            <label className="ml-1 block text-sm font-medium text-gray-700">
              About Me
            </label>
            <textarea
              name="about_me"
              value={formData.about_me}
              onChange={handleChange}
              placeholder="Tell people a little about yourself"
              rows={4}
              className="w-full rounded-bento border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder:text-gray-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            />
          </div>

          <div className="space-y-2">
            <label className="ml-1 block text-sm font-medium text-gray-700">
              Avatar / Image
            </label>
            <input
              type="file"
              accept="image/*"
              onChange={handleAvatarChange}
              className="w-full rounded-bento border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 file:mr-3 file:rounded-bento file:border-0 file:bg-gray-100 file:px-3 file:py-2 file:text-sm file:font-medium"
            />
            {avatarPreviewUrl ? (
              <div className="flex items-center gap-3 rounded-bento border border-gray-200 bg-gray-50 p-3">
                <img
                  src={avatarPreviewUrl}
                  alt="Avatar preview"
                  className="h-14 w-14 rounded-full object-cover"
                />
                <p className="text-sm text-gray-500">Avatar preview</p>
              </div>
            ) : null}
          </div>

          {error && (
            <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-sm text-red-600">
              {error}
            </div>
          )}

          {success && (
            <div className="rounded-lg bg-green-50 border border-green-200 p-3 text-sm text-green-600">
              {success}
            </div>
          )}

          <Button
            type="submit"
            variant="primary"
            className="w-full"
            disabled={isLoading}
          >
            {isLoading
              ? 'Creating Account...'
              : 'Create Account'}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-500">
          Already have an account?{' '}
          <Link
            href={nextPath === '/feed' ? '/login' : `/login?next=${encodeURIComponent(nextPath)}`}
            className="text-indigo-600 hover:underline"
          >
            Sign in
          </Link>
        </p>

      </div>
    </div>
  );
}
