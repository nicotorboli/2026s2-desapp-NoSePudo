import axios from 'axios';

// HealthResponseDTO mirrors backend DTO per Constitution Principle V
export interface HealthResponseDTO {
  status: string;
  timestamp: string;
}

const client = axios.create({
  baseURL: '/api',
});

// Private HTTP helper per Constitution Principle V
const get = async <T>(url: string): Promise<T> => {
  const response = await client.get<T>(url);
  return response.data;
};

// Exported endpoint function per Constitution Principle V
export const getHealth = async (): Promise<HealthResponseDTO> => {
  return get<HealthResponseDTO>('/health');
};
