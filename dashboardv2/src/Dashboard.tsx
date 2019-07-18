import * as React from 'react';
import { BrowserRouter as Router, Switch, Route } from 'react-router-dom';

import { Grommet, Box, Paragraph, Heading } from 'grommet';
import { aruba } from 'grommet-theme-aruba';

import useRouter from './useRouter';
import Split from './Split';
import Loading from './Loading';
import AppsListNav from './AppsListNav';
import ExternalAnchor from './ExternalAnchor';
import { DisplayErrors } from './useErrorHandler';

// DEBUG:
import { default as client, Client } from './client';
declare global {
	interface Window {
		client: Client;
	}
}
if (typeof window !== 'undefined') {
	window.client = client;
}

const AppComponent = React.lazy(() => import('./AppComponent'));

function appNameFromPath(path: string): string {
	const m = path.match(/\/apps\/[^/]+/);
	return m ? m[0].slice(1) : '';
}

/*
 * <Dashboard> is the root component of the dashboard app
 */
function DashboardInner() {
	const { location } = useRouter();
	const currentPath = React.useMemo(() => location.pathname || '', [location.pathname]);
	const [appName, setAppName] = React.useState<string>(appNameFromPath(currentPath));
	React.useEffect(
		() => {
			setAppName(appNameFromPath(currentPath));
		},
		[currentPath]
	);

	return (
		<Split>
			<Box tag="aside" basis="medium" flex={false} background="neutral-1" fill>
				<Box tag="header" pad="small">
					<h1>Flynn Dashboard</h1>
				</Box>
				<Box flex>
					<AppsListNav />
				</Box>
				<Box tag="footer" alignSelf="center">
					<Paragraph size="small" margin="xsmall">
						Flynn is designed, built, and managed by Prime Directive, Inc.
						<br />
						&copy; 2013-
						{new Date().getFullYear()} Prime Directive, Inc. Flynn® is a trademark of Prime Directive, Inc.
					</Paragraph>
					<Paragraph size="small" margin="xsmall">
						<ExternalAnchor href="https://flynn.io/legal/privacy">Privacy Policy</ExternalAnchor>
						&nbsp;|&nbsp;
						<ExternalAnchor href="https://flynn.io/docs/trademark-guidelines">Trademark Guidelines</ExternalAnchor>
					</Paragraph>
				</Box>
			</Box>

			<Box pad="xsmall" fill overflow="scroll" gap="small">
				<DisplayErrors />
				<React.Suspense fallback={<Loading />}>
					<Switch>
						<Route path="/apps/:appID">
							<AppComponent key={appName} name={appName} />
						</Route>
						<Route path="/">
							<Heading>Select an app to begin.</Heading>
						</Route>
					</Switch>
				</React.Suspense>
			</Box>
		</Split>
	);
}

const greenHexColor = '#1BB45E';
const modifiedAruba = Object.assign({}, aruba, {
	global: Object.assign({}, (aruba as any).global, {
		colors: Object.assign({}, (aruba as any).global.colors, {
			brand: greenHexColor,
			control: Object.assign({}, (aruba as any).global.colors.control, {
				light: greenHexColor
			})
		})
	})
});

export default function Dashboard() {
	return (
		<Grommet full theme={modifiedAruba} cssVars>
			<Router>
				<DashboardInner />
			</Router>
		</Grommet>
	);
}
