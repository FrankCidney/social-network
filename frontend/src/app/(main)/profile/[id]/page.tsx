'use client';

import { useParams } from 'next/navigation';
import { ProfilePageView } from '@/components/profile/ProfilePageView';

export default function UserProfilePage() {
  const params = useParams<{ id: string }>();
  const userId = Array.isArray(params.id) ? params.id[0] : params.id;

  return <ProfilePageView userId={userId} />;
}
