export type RouteValue = (typeof ROUTES)[keyof typeof ROUTES];

export interface UserProfile {
  me: {
    id: string;
    name: string;
    email: string;
    role: string;
    status: string;
    createdAt: string;
  };
}
