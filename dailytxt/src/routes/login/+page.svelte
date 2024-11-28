<script>
	import img from '$lib/assets/locked_heart_with_keyhole.svg';
</script>

<div class="logo-login-flex d-flex justify-content-center align-items-center flex-row h-100">
	<div class="logo-wrapper d-flex flex-column align-items-center">
		<img id="largeLogo" src={img} alt="locked heart with keyhole" />
		<p>DailyTxT</p>
	</div>
	<div class="login-wrapper">
		<div class="accordion" id="loginAccordion">
			<div class="accordion-item">
				<h2 class="accordion-header">
					<button
						class="accordion-button"
						type="button"
						data-bs-toggle="collapse"
						data-bs-target="#collapseOne"
						aria-expanded="true"
						aria-controls="collapseOne"
					>
						Login
					</button>
				</h2>
				<div
					id="collapseOne"
					class="accordion-collapse collapse show"
					data-bs-parent="#loginAccordion"
				>
					<div class="accordion-body">
						<form action="">
							<div class="form-floating mb-3">
								<input type="text" class="form-control" id="loginUsername" placeholder="Username" />
								<label for="loginUsername">Username</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="loginPassword"
									placeholder="Password"
								/>
								<label for="loginPassword">Password</label>
							</div>
							<div class="d-flex justify-content-center">
								<button type="submit" class="btn btn-primary">Login</button>
							</div>
						</form>
					</div>
				</div>
			</div>
			<div class="accordion-item">
				<h2 class="accordion-header">
					<button
						class="accordion-button collapsed"
						type="button"
						data-bs-toggle="collapse"
						data-bs-target="#collapseTwo"
						aria-expanded="false"
						aria-controls="collapseTwo"
					>
						Registrierung
					</button>
				</h2>
				<div id="collapseTwo" class="accordion-collapse collapse" data-bs-parent="#loginAccordion">
					<div class="accordion-body">
						<form action="">
							<div class="form-floating mb-3">
								<input
									type="text"
									class="form-control"
									id="registerUsername"
									placeholder="Username"
								/>
								<label for="registerUsername">Username</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="registerPassword"
									placeholder="Password"
								/>
								<label for="registerPassword">Password</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="registerPassword2"
									placeholder="Password bestätigen"
								/>
								<label for="registerPassword2">Password bestätigen</label>
							</div>
							<div class="d-flex justify-content-center">
								<button type="submit" class="btn btn-primary">Registrieren</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.logo-wrapper {
		width: 50%;
	}

	#largeLogo {
		width: 40%;
		min-height: 10%;
	}

	.login-wrapper {
		width: 50%;
	}

	#loginAccordion {
		width: 70%;
	}

	@media screen and (max-width: 768px) {
		.logo-login-flex {
			flex-direction: column !important;
		}

		.login-wrapper {
			min-width: 50%;
			max-width: 75%;
		}
	}
</style>
